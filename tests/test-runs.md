# Test case scenarios

### Basic "help" options tests

```
./gh-repo-transfer --help
./gh-repo-transfer deps --help   
./gh-repo-transfer transfer --help
./gh-repo-transfer archive --help
```

---

### This is a basic dependency option test. WITHOUT validation against a target ORG

#### Check dependencies for a repo

```
time ./gh-repo-transfer deps \
github-innersource/gh-repo-inspect-test-main \
github-innersource/gh-repo-inspect-test-sub
```

---

### This is a basic dependency option test. NO validation against a target ORG

#### Check dependencies for a repo

```
time ./gh-repo-transfer deps \
github-innersource/gh-repo-inspect-test-main -t jester-lab
```

---

### These tests DO NOT enforce the transfer option, therefore they will fail if the dependency conditions are not all met

#### Transfer from A to B - NO team assignement

```
./gh-repo-transfer transfer jester-lab/gh-repo-inspect-test-main -t github-innersource
```

#### Transfer from B to A - NO team assignement

```
./gh-repo-transfer transfer github-innersource/gh-repo-inspect-test-main -t jester-lab
```

---

### These tests DO NOT enforce the transfer option, therefore they will fail if the dependency conditions are not all met

#### Transfer from A to B and assign teams, without enforcing the creation of teams that do not exist in the target ORG (--assign)

```
./gh-repo-transfer transfer jester-lab/gh-repo-inspect-test-main -t github-innersource --assign
```

#### Transfer from B to A and assign teams, without enforcing the creation of teams that do not exist in the target ORG (--assign)

```
./gh-repo-transfer transfer github-innersource/gh-repo-inspect-test-main -t jester-lab --assign
```

---

### These tests DO enforce the transfer option, therefore the transfer should always succeed regardless of the dependency conditions

#### Transfer from A to B and assign teams, WITHOUT enforcing the creation of teams that do not exist in the target ORG (--assign --enforce)

```
./gh-repo-transfer transfer jester-lab/gh-repo-inspect-test-main -t github-innersource --assign 
--enforce
```

#### Transfer from B to A and assign teams, WITHOUT enforcing the creation of teams that do not exist in the target ORG (--assign --enforce)

```
./gh-repo-transfer transfer github-innersource/gh-repo-inspect-test-main -t jester-lab --assign --enforce
```

---

### These tests DO enforce the transfer option, therefore the transfer should always succeed regardless of the dependency conditions. 
### In addition, the creation of teams that do not exist in the target ORG should also be enforced.

#### Transfer from A to B and assign teams, WITH enforcing the creation of teams that do not exist in the target ORG (--assign --create --enforce)

```
./gh-repo-transfer transfer jester-lab/gh-repo-inspect-test-main -t github-innersource --assign --create --enforce
```

#### Transfer from B to A and assign teams, WITH enforcing the creation of teams that do not exist in the target ORG (--assign --create --enforce)

```
./gh-repo-transfer transfer github-innersource/gh-repo-inspect-test-main -t jester-lab --assign --create --enforce
```

---

## Archive Repo from ORG `A` to ORG `B`

```
./gh-repo-transfer archive github-innersource/gh-repo-inspect-test-main -t jester-lab 
```