#include "testlib.h"
using namespace std;

bool spaces(string s) {
	int len = s.length();
	for (int i = 0; i < len; i++)
		if (s[i] != ' ')
			return false;
	return true;
}

bool compare(string a, string b) {
	int i, alen = a.length(), blen = b.length();
	for (i = 0; i < alen && i < blen; i++)
		if (a[i] != b[i])
			return false;
	while (i < alen)
		if (a[i++] != ' ')
			return false;
	while (i < blen)
		if (b[i++] != ' ')
			return false;
	return true;
}

int main(int argc, char * argv[]) {
	setName("Answer checker, ignore trailing spaces and empty lines");
	registerTestlibCmd(argc, argv);

	int line = 0;
	while (!ouf.seekEof() && !ans.seekEof())
		if (line++, !compare(ouf.readLine(), ans.readLine()))
			quitf(_wa, "at line %d", line);

	while (!ouf.seekEof())
		if (line++, !spaces(ouf.readLine()))
			quitf(_wa, "at line %d", line);

	while (!ans.seekEof())
		if (line++, !spaces(ans.readLine()))
			quitf(_wa, "at line %d", line);

	quitf(_ok, "- %d line(s) in total", line);
}
