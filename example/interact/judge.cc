#include <bits/stdc++.h>

using namespace std;

enum { AC, WA, PE, TLE, MLE, OLE, CE, RE, SE };
auto casenv = getenv("case");
int cas = casenv ? atoi(casenv) : 0;

int main() {
  srand(time(0));
  int m = pow(10, cas + 1);
  int a = rand() % m, b = rand() % m;
  cout << a << " " << b << endl;
  cin >> m;
  return a + b == m ? AC : WA;
}
