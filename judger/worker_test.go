package judger

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRunOneLeasesExecutesAndPostsResult(t *testing.T) {
	if os.Getenv("DOCKER_TEST") != "1" {
		t.Skip("set DOCKER_TEST=1 to run worker execution test")
	}
	runner := buildRunner(t)
	work := t.TempDir()
	input := filepath.Join(work, "1.in")
	answer := filepath.Join(work, "1.out")
	if err := os.WriteFile(input, []byte("42\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(answer, []byte("42\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	gotResult := make(chan resultRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			http.Error(w, "bad token", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/api/judger/lease":
			_ = json.NewEncoder(w).Encode(leaseResponse{Task: &leaseTask{
				ID:           7,
				SubmissionID: 11,
				Attempt:      2,
				Source:       "cat\n",
				Lang:         testShellLang(),
				Mode:         ModeDefault,
				Limits:       Limits{TimeMS: 1000, OutputKB: 64},
				Cases: []casePayload{{
					ID:     "1",
					Input:  input,
					Answer: answer,
					Score:  100,
				}},
			}})
		case "/api/judger/tasks/7/result":
			var req resultRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Error(err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			gotResult <- req
			w.WriteHeader(http.StatusAccepted)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	worked, err := RunOne(ctx, WorkerConfig{
		Server: server.URL,
		Token:  "secret",
		Runner: runner,
		Work:   filepath.Join(work, "jobs"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !worked {
		t.Fatal("expected task")
	}

	select {
	case result := <-gotResult:
		if result.SubmissionID != 11 || result.Attempt != 2 || result.Status != string(VerdictAccepted) || result.Score != 100 {
			t.Fatalf("result = %#v", result)
		}
		if len(result.Cases) != 1 || result.Cases[0].Status != string(VerdictAccepted) {
			t.Fatalf("cases = %#v", result.Cases)
		}
	case <-ctx.Done():
		t.Fatal("missing result callback")
	}
}

func TestRunOneDownloadsAssetsForRelativeCases(t *testing.T) {
	if os.Getenv("DOCKER_TEST") != "1" {
		t.Skip("set DOCKER_TEST=1 to run worker execution test")
	}
	runner := buildRunner(t)
	work := t.TempDir()

	gotResult := make(chan resultRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			http.Error(w, "bad token", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/api/judger/lease":
			_ = json.NewEncoder(w).Encode(leaseResponse{Task: &leaseTask{
				ID:           8,
				SubmissionID: 12,
				Attempt:      1,
				Source:       "cat\n",
				Lang:         testShellLang(),
				Mode:         ModeDefault,
				Limits:       Limits{TimeMS: 1000, OutputKB: 64},
				Cases: []casePayload{{
					ID:     "1",
					Input:  "data/1.in",
					Answer: "data/1.out",
					Score:  100,
				}},
			}})
		case "/api/judger/tasks/8/assets.zip":
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(testAssetZip(t))
		case "/api/judger/tasks/8/result":
			var req resultRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Error(err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			gotResult <- req
			w.WriteHeader(http.StatusAccepted)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	worked, err := RunOne(ctx, WorkerConfig{
		Server: server.URL,
		Token:  "secret",
		Runner: runner,
		Work:   filepath.Join(work, "jobs"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !worked {
		t.Fatal("expected task")
	}

	select {
	case result := <-gotResult:
		if result.SubmissionID != 12 || result.Attempt != 1 || result.Status != string(VerdictAccepted) || result.Score != 100 {
			t.Fatalf("result = %#v", result)
		}
	case <-ctx.Done():
		t.Fatal("missing result callback")
	}
}

func TestExtractTaskAssetsRejectsZipSlip(t *testing.T) {
	for _, name := range []string{"../escape.txt", "/absolute.txt", "data/../../escape.txt"} {
		t.Run(name, func(t *testing.T) {
			var body bytes.Buffer
			writer := zip.NewWriter(&body)
			file, err := writer.Create(name)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := file.Write([]byte("bad")); err != nil {
				t.Fatal(err)
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			if err := extractTaskAssets(body.Bytes(), t.TempDir()); err == nil {
				t.Fatal("expected zip-slip asset to be rejected")
			}
		})
	}
}

func testAssetZip(t *testing.T) []byte {
	t.Helper()
	var body bytes.Buffer
	writer := zip.NewWriter(&body)
	for name, content := range map[string]string{
		"data/1.in":  "42\n",
		"data/1.out": "42\n",
	} {
		file, err := writer.Create(name)
		if err != nil {
			t.Fatalf("create zip file failed: %v", err)
		}
		if _, err := file.Write([]byte(content)); err != nil {
			t.Fatalf("write zip file failed: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip failed: %v", err)
	}
	return body.Bytes()
}
