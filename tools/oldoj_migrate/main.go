package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/doveccl/doj/utils"
)

type oldProblem struct {
	OldID       string   `json:"oldId"`
	ID          uint     `json:"id"`
	Title       string   `json:"title"`
	Tags        []string `json:"tags"`
	TimeLimit   float64  `json:"timeLimit"`
	MemoryLimit float64  `json:"memoryLimit"`
	Data        string   `json:"data"`
	Content     string   `json:"content"`
}

type targetProblem struct {
	ID    uint
	Title string
	Cases int
}

type migrationPlan struct {
	Candidates []oldProblem
	Skip       []oldProblem
	Cleanup    []targetProblem
}

var titlePrefix = regexp.MustCompile(`^P[0-9]+\s*-\s*`)

var errNoCases = errors.New("zip has no input/output case pairs")

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("usage: oldoj_migrate <plan|migrate|cleanup>")
	}
	switch os.Args[1] {
	case "plan":
		return planCommand(os.Args[2:])
	case "migrate":
		return migrateCommand(os.Args[2:])
	case "cleanup":
		return cleanupCommand(os.Args[2:])
	case "verify":
		return verifyCommand(os.Args[2:])
	default:
		return fmt.Errorf("unknown command %q", os.Args[1])
	}
}

func planCommand(args []string) error {
	fs := flag.NewFlagSet("plan", flag.ExitOnError)
	sshHost := fs.String("ssh", "51cspnoip.cn", "SSH host that can access old Mongo and test Postgres")
	limit := fs.Int("limit", 100, "number of real data problems to select")
	scan := fs.Int("scan", 320, "number of old numeric P problems to scan")
	timeout := fs.Duration("timeout", 20*time.Second, "timeout for each SSH-backed metadata query")
	manifestPath := fs.String("manifest", "", "optional local manifest JSONL")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	oldRows, err := loadOldProblems(ctx, *sshHost, *manifestPath, *scan)
	if err != nil {
		return err
	}
	targetRows, err := loadTargetProblems(ctx, *sshHost)
	if err != nil {
		return err
	}
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		return err
	}
	targetCases := map[uint]int{}
	for id := range targetRows {
		cases, err := countStoredCases(ctx, store, id)
		if err != nil {
			return err
		}
		row := targetRows[id]
		row.Cases = cases
		targetRows[id] = row
		targetCases[id] = cases
	}
	plan := buildPlan(oldRows, targetRows, targetCases, *limit)
	printPlan(plan)
	return nil
}

func migrateCommand(args []string) error {
	fs := flag.NewFlagSet("migrate", flag.ExitOnError)
	sshHost := fs.String("ssh", "51cspnoip.cn", "SSH host that can access old Mongo and test Postgres")
	limit := fs.Int("limit", 5, "maximum problems to migrate in this batch")
	scan := fs.Int("scan", 320, "number of old numeric P problems to scan")
	sleep := fs.Duration("sleep", 2*time.Second, "pause between migrated problems")
	timeout := fs.Duration("timeout", 30*time.Second, "timeout for each SSH-backed operation")
	apply := fs.Bool("apply", false, "perform writes; default is dry-run")
	manifestPath := fs.String("manifest", "", "optional local manifest JSONL")
	if err := fs.Parse(args); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	oldRows, err := loadOldProblems(ctx, *sshHost, *manifestPath, *scan)
	if err != nil {
		return err
	}
	targetRows, err := loadTargetProblems(ctx, *sshHost)
	if err != nil {
		return err
	}
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		return err
	}
	targetCases := map[uint]int{}
	for id := range targetRows {
		cases, err := countStoredCases(ctx, store, id)
		if err != nil {
			return err
		}
		targetCases[id] = cases
	}
	plan := buildPlan(oldRows, targetRows, targetCases, math.MaxInt)
	count := 0
	for _, item := range plan.Candidates {
		if count >= *limit {
			break
		}
		fmt.Printf("migrate P%d %s\n", item.ID, cleanTitle(item))
		if *apply {
			itemCtx, cancel := context.WithTimeout(context.Background(), *timeout)
			err := migrateOne(itemCtx, *sshHost, store, item)
			cancel()
			if err != nil {
				if errors.Is(err, errNoCases) {
					fmt.Printf("skip P%d: %v\n", item.ID, err)
					continue
				}
				return fmt.Errorf("migrate P%d: %w", item.ID, err)
			}
			time.Sleep(*sleep)
		}
		count++
	}
	if !*apply {
		fmt.Println("dry-run only; add --apply to write")
	}
	return nil
}

func cleanupCommand(args []string) error {
	fs := flag.NewFlagSet("cleanup", flag.ExitOnError)
	sshHost := fs.String("ssh", "51cspnoip.cn", "SSH host that can access test Postgres")
	apply := fs.Bool("apply", false, "perform soft deletes; default is dry-run")
	maxID := fs.Uint("max-id", 7999, "only consider local problem IDs up to this value")
	timeout := fs.Duration("timeout", 20*time.Second, "timeout for each SSH-backed query")
	if err := fs.Parse(args); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	targetRows, err := loadTargetProblems(ctx, *sshHost)
	if err != nil {
		return err
	}
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		return err
	}
	var cleanup []targetProblem
	for id, row := range targetRows {
		if id > uint(*maxID) {
			continue
		}
		cases, err := countStoredCases(ctx, store, id)
		if err != nil {
			return err
		}
		if cases == 0 {
			row.Cases = cases
			cleanup = append(cleanup, row)
		}
	}
	sort.Slice(cleanup, func(i, j int) bool { return cleanup[i].ID < cleanup[j].ID })
	for _, row := range cleanup {
		fmt.Printf("cleanup P%d %s\n", row.ID, row.Title)
	}
	if !*apply {
		fmt.Println("dry-run only; add --apply to soft-delete")
		return nil
	}
	for _, row := range cleanup {
		if err := softDeleteProblem(ctx, *sshHost, row.ID); err != nil {
			return err
		}
	}
	return nil
}

func verifyCommand(args []string) error {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	sshHost := fs.String("ssh", "51cspnoip.cn", "SSH host that can access test Postgres")
	minData := fs.Int("min-data", 100, "minimum active problems with real data cases")
	maxEmpty := fs.Int("max-empty", 0, "maximum active problems without data cases")
	timeout := fs.Duration("timeout", 20*time.Second, "timeout for SSH-backed queries")
	if err := fs.Parse(args); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	targetRows, err := loadTargetProblems(ctx, *sshHost)
	if err != nil {
		return err
	}
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		return err
	}
	report, err := verifyTargetProblems(ctx, store, targetRows)
	if err != nil {
		return err
	}
	fmt.Printf("active=%d with_data=%d without_data=%d\n", report.Active, report.WithData, report.WithoutData)
	if len(report.Empty) > 0 {
		fmt.Println("empty active problems:")
		for _, row := range report.Empty {
			fmt.Printf("P%d %s\n", row.ID, row.Title)
		}
	}
	if report.WithData < *minData {
		return fmt.Errorf("with_data=%d, want at least %d", report.WithData, *minData)
	}
	if report.WithoutData > *maxEmpty {
		return fmt.Errorf("without_data=%d, want at most %d", report.WithoutData, *maxEmpty)
	}
	return nil
}

type verifyReport struct {
	Active      int
	WithData    int
	WithoutData int
	Empty       []targetProblem
}

func verifyTargetProblems(ctx context.Context, store utils.ObjectStore, targetRows map[uint]targetProblem) (verifyReport, error) {
	report := verifyReport{Active: len(targetRows)}
	for id, row := range targetRows {
		cases, err := countStoredCases(ctx, store, id)
		if err != nil {
			return verifyReport{}, err
		}
		row.Cases = cases
		if cases > 0 {
			report.WithData++
			continue
		}
		report.WithoutData++
		report.Empty = append(report.Empty, row)
	}
	sort.Slice(report.Empty, func(i, j int) bool { return report.Empty[i].ID < report.Empty[j].ID })
	return report, nil
}

func loadOldProblems(ctx context.Context, sshHost string, manifestPath string, scan int) ([]oldProblem, error) {
	if manifestPath != "" {
		return readManifest(manifestPath)
	}
	script := fmt.Sprintf(`const cur = db.problems.aggregate([
  { $match: { "contest.key": { $regex: /^[0-9]+$/ } } },
  { $addFields: { keyNum: { $toInt: "$contest.key" } } },
  { $sort: { keyNum: -1 } },
  { $limit: %d },
]);
for (const p of cur) {
  print(JSON.stringify({
    oldId: p._id.valueOf(),
    id: Number(p.contest.key),
    title: p.title || "",
    tags: p.tags || [],
    timeLimit: p.timeLimit || 1,
    memoryLimit: p.memoryLimit || 134217728,
    data: p.data ? p.data.valueOf() : "",
    content: p.content || ""
  }));
}`, scan)
	out, err := runSSH(ctx, sshHost, "docker", "exec", "-i", "mongo", "mongosh", "--quiet", "doj", "--file", "/dev/stdin", stdin(script))
	if err != nil {
		return nil, err
	}
	return parseOldProblems(out)
}

func readManifest(file string) ([]oldProblem, error) {
	raw, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	return parseOldProblems(string(raw))
}

func parseOldProblems(raw string) ([]oldProblem, error) {
	var rows []oldProblem
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var row oldProblem
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func loadTargetProblems(ctx context.Context, sshHost string) (map[uint]targetProblem, error) {
	sql := `select id || E'\t' || title from problems where deleted_at is null order by id;`
	out, err := runSSH(ctx, sshHost, "docker", "exec", "-i", "doj-test-postgres-1", "psql", "-h", "127.0.0.1", "-U", "doj", "-d", "doj", "-At", "-c", sql)
	if err != nil {
		return nil, err
	}
	rows := map[uint]targetProblem{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		id64, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			return nil, err
		}
		rows[uint(id64)] = targetProblem{ID: uint(id64), Title: parts[1]}
	}
	return rows, nil
}

func buildPlan(oldRows []oldProblem, targetRows map[uint]targetProblem, targetCases map[uint]int, limit int) migrationPlan {
	var plan migrationPlan
	seenTitle := map[string]bool{}
	for _, row := range targetRows {
		if targetCases[row.ID] > 0 {
			seenTitle[normalizeTitle(row.Title)] = true
		}
	}
	for _, row := range oldRows {
		if len(plan.Candidates) >= limit {
			break
		}
		if row.Data == "" {
			continue
		}
		title := normalizeTitle(cleanTitle(row))
		if targetCases[row.ID] > 0 || seenTitle[title] {
			plan.Skip = append(plan.Skip, row)
			continue
		}
		plan.Candidates = append(plan.Candidates, row)
	}
	for _, row := range targetRows {
		if targetCases[row.ID] == 0 && row.ID < 8000 {
			plan.Cleanup = append(plan.Cleanup, row)
		}
	}
	sort.Slice(plan.Cleanup, func(i, j int) bool { return plan.Cleanup[i].ID < plan.Cleanup[j].ID })
	return plan
}

func printPlan(plan migrationPlan) {
	fmt.Printf("candidates=%d skipped=%d cleanup=%d\n", len(plan.Candidates), len(plan.Skip), len(plan.Cleanup))
	for _, row := range plan.Candidates {
		fmt.Printf("candidate P%d %s data=%s\n", row.ID, cleanTitle(row), row.Data)
	}
	if len(plan.Cleanup) > 0 {
		fmt.Println("cleanup candidates:")
		for _, row := range plan.Cleanup {
			fmt.Printf("cleanup P%d %s\n", row.ID, row.Title)
		}
	}
}

func migrateOne(ctx context.Context, sshHost string, store utils.ObjectStore, row oldProblem) error {
	data, err := streamOldZip(ctx, sshHost, row.Data)
	if err != nil {
		return err
	}
	files, cases, err := zipDataFiles(data)
	if err != nil {
		return err
	}
	if cases == 0 {
		return errNoCases
	}
	for name, body := range files {
		key := path.Join("problems", strconv.Itoa(int(row.ID)), "data", name)
		if err := store.Put(ctx, key, bytes.NewReader(body), int64(len(body)), "application/octet-stream"); err != nil {
			return err
		}
	}
	statement := strings.TrimSpace(row.Content)
	if statement == "" {
		statement = "# " + cleanTitle(row)
	}
	if err := store.Put(ctx, path.Join("problems", strconv.Itoa(int(row.ID)), "statement.md"), strings.NewReader(statement), int64(len(statement)), "text/markdown; charset=utf-8"); err != nil {
		return err
	}
	return upsertProblem(ctx, sshHost, row)
}

func streamOldZip(ctx context.Context, sshHost string, dataID string) ([]byte, error) {
	script := fmt.Sprintf(`const chunks = db.fs.chunks.find({ files_id: ObjectId(%q) }).sort({ n: 1 }).toArray();
const buf = Buffer.concat(chunks.map((c) => Buffer.from(c.data.buffer)));
process.stdout.write(buf);`, dataID)
	return runSSHBytes(ctx, sshHost, "docker", "exec", "-i", "mongo", "mongosh", "--quiet", "doj", "--file", "/dev/stdin", stdin(script))
}

func zipDataFiles(data []byte) (map[string][]byte, int, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, 0, err
	}
	files := map[string][]byte{}
	inputs := map[string]bool{}
	outputs := map[string]bool{}
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		name := strings.TrimLeft(path.Clean(strings.ReplaceAll(file.Name, "\\", "/")), "/")
		if name == "." || strings.HasPrefix(name, "../") {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return nil, 0, err
		}
		body, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			return nil, 0, err
		}
		files[name] = body
		stem, kind := utils.DataCaseStem(name)
		if kind == "in" {
			inputs[stem] = true
		}
		if kind == "out" {
			outputs[stem] = true
		}
	}
	cases := 0
	for stem := range inputs {
		if outputs[stem] {
			cases++
		}
	}
	return files, cases, nil
}

func upsertProblem(ctx context.Context, sshHost string, row oldProblem) error {
	tags, _ := json.Marshal(row.Tags)
	sql := fmt.Sprintf(`insert into problems (id,title,tags,visible,mode,time_ms,memory_mb,created_at,updated_at)
values (%d, %s, %s::jsonb, true, 'default', %d, %d, now(), now())
on conflict (id) do update set title=excluded.title, tags=excluded.tags, visible=true, mode=excluded.mode, time_ms=excluded.time_ms, memory_mb=excluded.memory_mb, deleted_at=null, updated_at=now();`,
		row.ID,
		sqlString(cleanTitle(row)),
		sqlString(string(tags)),
		int(math.Ceil(row.TimeLimit*1000)),
		memoryMB(row.MemoryLimit),
	)
	sql += "\nselect setval(pg_get_serial_sequence('problems','id'), greatest((select max(id) from problems), 1000), true);"
	_, err := runSSH(ctx, sshHost, "docker", "exec", "-i", "doj-test-postgres-1", "psql", "-h", "127.0.0.1", "-U", "doj", "-d", "doj", "-v", "ON_ERROR_STOP=1", "-c", sql)
	return err
}

func softDeleteProblem(ctx context.Context, sshHost string, id uint) error {
	sql := fmt.Sprintf("update problems set deleted_at=now(), updated_at=now() where id=%d and deleted_at is null;", id)
	_, err := runSSH(ctx, sshHost, "docker", "exec", "-i", "doj-test-postgres-1", "psql", "-h", "127.0.0.1", "-U", "doj", "-d", "doj", "-v", "ON_ERROR_STOP=1", "-c", sql)
	return err
}

func countStoredCases(ctx context.Context, store utils.ObjectStore, id uint) (int, error) {
	objects, err := store.List(ctx, path.Join("problems", strconv.Itoa(int(id)), "data"))
	if err != nil {
		return 0, err
	}
	inputs := map[string]bool{}
	outputs := map[string]bool{}
	prefix := path.Join("problems", strconv.Itoa(int(id)), "data") + "/"
	for _, object := range objects {
		name := strings.TrimPrefix(object.Key, prefix)
		stem, kind := utils.DataCaseStem(name)
		if kind == "in" {
			inputs[stem] = true
		}
		if kind == "out" {
			outputs[stem] = true
		}
	}
	cases := 0
	for stem := range inputs {
		if outputs[stem] {
			cases++
		}
	}
	return cases, nil
}

func cleanTitle(row oldProblem) string {
	title := strings.TrimSpace(row.Title)
	title = titlePrefix.ReplaceAllString(title, "")
	if title == "" {
		title = fmt.Sprintf("P%d", row.ID)
	}
	return title
}

func normalizeTitle(title string) string {
	return strings.ToLower(strings.Join(strings.Fields(titlePrefix.ReplaceAllString(strings.TrimSpace(title), "")), " "))
}

func memoryMB(bytesValue float64) int {
	if bytesValue <= 0 {
		return 256
	}
	return int(math.Ceil(bytesValue / 1024 / 1024))
}

func sqlString(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

type stdin string

func runSSH(ctx context.Context, host string, args ...any) (string, error) {
	out, err := runSSHBytes(ctx, host, args...)
	return string(out), err
}

func runSSHBytes(ctx context.Context, host string, args ...any) ([]byte, error) {
	var stdinReader io.Reader
	var command []string
	for _, arg := range args {
		switch value := arg.(type) {
		case stdin:
			stdinReader = strings.NewReader(string(value))
		case string:
			command = append(command, value)
		default:
			return nil, fmt.Errorf("unsupported ssh arg %T", arg)
		}
	}
	if len(command) == 0 {
		return nil, errors.New("ssh command is required")
	}
	cmd := exec.CommandContext(ctx, "ssh", host, shellCommand(command))
	if stdinReader != nil {
		cmd.Stdin = stdinReader
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		sum := sha1.Sum(out)
		return nil, fmt.Errorf("ssh %s failed: %w: %s output_sha1=%s", host, err, strings.TrimSpace(stderr.String()), hex.EncodeToString(sum[:]))
	}
	return out, nil
}

func shellCommand(args []string) string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		quoted = append(quoted, shellQuote(arg))
	}
	return strings.Join(quoted, " ")
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
