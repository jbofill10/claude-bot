package db

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) (*sql.DB, *Queries) {
	t.Helper()
	database, err := sql.Open("sqlite3", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := migrate(database); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database, NewQueries(database)
}

func TestRepoDeployScriptDefaultEmpty(t *testing.T) {
	_, q := setupTestDB(t)

	user, err := q.CreateUser("alice")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	repo, err := q.CreateRepo(user.ID, "myrepo", "/tmp/myrepo")
	if err != nil {
		t.Fatalf("create repo: %v", err)
	}

	if repo.DeployScript != "" {
		t.Errorf("expected empty deploy_script, got %q", repo.DeployScript)
	}
}

func TestUpdateRepoDeployScript(t *testing.T) {
	_, q := setupTestDB(t)

	user, err := q.CreateUser("bob")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	repo, err := q.CreateRepo(user.ID, "myrepo", "/tmp/myrepo")
	if err != nil {
		t.Fatalf("create repo: %v", err)
	}

	err = q.UpdateRepoDeployScript(repo.ID, "./deploy.sh")
	if err != nil {
		t.Fatalf("update deploy script: %v", err)
	}

	updated, err := q.GetRepo(repo.ID)
	if err != nil {
		t.Fatalf("get repo: %v", err)
	}

	if updated.DeployScript != "./deploy.sh" {
		t.Errorf("expected deploy_script %q, got %q", "./deploy.sh", updated.DeployScript)
	}
}

func TestUpdateRepoDeployScriptClear(t *testing.T) {
	_, q := setupTestDB(t)

	user, err := q.CreateUser("charlie")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	repo, err := q.CreateRepo(user.ID, "myrepo", "/tmp/myrepo")
	if err != nil {
		t.Fatalf("create repo: %v", err)
	}

	// Set then clear
	_ = q.UpdateRepoDeployScript(repo.ID, "./deploy.sh")
	err = q.UpdateRepoDeployScript(repo.ID, "")
	if err != nil {
		t.Fatalf("clear deploy script: %v", err)
	}

	updated, err := q.GetRepo(repo.ID)
	if err != nil {
		t.Fatalf("get repo: %v", err)
	}

	if updated.DeployScript != "" {
		t.Errorf("expected empty deploy_script after clear, got %q", updated.DeployScript)
	}
}

func TestListReposIncludesDeployScript(t *testing.T) {
	_, q := setupTestDB(t)

	user, err := q.CreateUser("dave")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	repo, err := q.CreateRepo(user.ID, "myrepo", "/tmp/myrepo")
	if err != nil {
		t.Fatalf("create repo: %v", err)
	}

	_ = q.UpdateRepoDeployScript(repo.ID, "make deploy")

	repos, err := q.ListRepos(user.ID)
	if err != nil {
		t.Fatalf("list repos: %v", err)
	}

	if len(repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(repos))
	}

	if repos[0].DeployScript != "make deploy" {
		t.Errorf("expected deploy_script %q in list, got %q", "make deploy", repos[0].DeployScript)
	}
}
