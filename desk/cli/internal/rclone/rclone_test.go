package rclone

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ElmerMosquera2/cssh/desk/cli/internal/profile"
)

func TestParseGitURL(t *testing.T) {
	tests := []struct {
		url          string
		expectedUser string
		expectedHost string
		expectedRepo string
	}{
		{
			url:          "git@github.com:ElmerMosquera2/cssh.git",
			expectedUser: "git",
			expectedHost: "github.com",
			expectedRepo: "cssh",
		},
		{
			url:          "gitlab.com:org/subgroup/project.git",
			expectedUser: "git",
			expectedHost: "gitlab.com",
			expectedRepo: "project",
		},
		{
			url:          "ssh://git@custom-host.com:2222/org/my-repo.git",
			expectedUser: "git",
			expectedHost: "custom-host.com",
			expectedRepo: "my-repo",
		},
		{
			url:          "git@bitbucket.org:team/repo",
			expectedUser: "git",
			expectedHost: "bitbucket.org",
			expectedRepo: "repo",
		},
	}

	for _, tc := range tests {
		info, err := ParseGitURL(tc.url)
		if err != nil {
			t.Errorf("ParseGitURL(%q) unexpected error: %v", tc.url, err)
			continue
		}
		if info.User != tc.expectedUser {
			t.Errorf("expected user %q, got %q", tc.expectedUser, info.User)
		}
		if info.Host != tc.expectedHost {
			t.Errorf("expected host %q, got %q", tc.expectedHost, info.Host)
		}
		if info.RepoName != tc.expectedRepo {
			t.Errorf("expected repo %q, got %q", tc.expectedRepo, info.RepoName)
		}
	}
}

func TestResolveProfile(t *testing.T) {
	cfg := &profile.Config{
		Default: "personal",
		Profiles: map[string]profile.Profile{
			"personal": {
				Key:   "~/.ssh/id_pers",
				Hosts: []string{"github.com"},
			},
			"empresa": {
				Key:   "~/.ssh/id_work",
				Hosts: []string{"gitlab.com"},
			},
			"ambiguo": {
				Key:   "~/.ssh/id_amb",
				Hosts: []string{"github.com"}, // Conflicto con personal en github.com
			},
			"servidor": {
				Key: "~/.ssh/id_vps",
			},
		},
	}

	// 1. Explícito gana
	p, err := ResolveProfile(cfg, "empresa", "git@github.com:user/repo.git")
	if err != nil {
		t.Fatalf("ResolveProfile explicit error: %v", err)
	}
	if p.Name != "empresa" {
		t.Errorf("expected explicit 'empresa', got %q", p.Name)
	}

	// 2. Match único: gitlab.com sólo reclamado por 'empresa'
	p, err = ResolveProfile(cfg, "", "git@gitlab.com:user/repo.git")
	if err != nil {
		t.Fatalf("ResolveProfile single match error: %v", err)
	}
	if p.Name != "empresa" {
		t.Errorf("expected match único 'empresa', got %q", p.Name)
	}

	// 3. Ambigüedad: github.com reclamado por 'personal' y 'ambiguo'
	_, err = ResolveProfile(cfg, "", "git@github.com:user/repo.git")
	if err == nil {
		t.Fatal("expected ambiguity error, got nil")
	}
	if !strings.Contains(err.Error(), "múltiples perfiles reclaman el host \"github.com\"") {
		t.Errorf("unexpected error message: %v", err)
	}

	// 4. Default: bitbucket.org no reclamado por ninguno -> usa default 'personal'
	p, err = ResolveProfile(cfg, "", "git@bitbucket.org:user/repo.git")
	if err != nil {
		t.Fatalf("ResolveProfile default fallback error: %v", err)
	}
	if p.Name != "personal" {
		t.Errorf("expected default fallback 'personal', got %q", p.Name)
	}

	// 5. Sin default y sin match -> error
	cfgNoDef := &profile.Config{
		Profiles: map[string]profile.Profile{
			"servidor": {Key: "~/.ssh/id_vps"},
		},
	}
	_, err = ResolveProfile(cfgNoDef, "", "git@bitbucket.org:user/repo.git")
	if err == nil {
		t.Fatal("expected error when no default and no host match, got nil")
	}
}

func TestExecuteDryRun(t *testing.T) {
	buf := new(bytes.Buffer)
	p := profile.Profile{
		Name:    "pers",
		Key:     "~/.ssh/id_pers",
		GitName: "Pers Name",
		Email:   "pers@mail.com",
	}

	err := Execute(CloneOptions{
		Profile:  p,
		URL:      "git@github.com:org/app.git",
		Into:     "mi-app",
		NoVerify: false,
		DryRun:   true,
		GitArgs:  []string{"--depth=1"},
		Stdout:   buf,
	})
	if err != nil {
		t.Fatalf("Execute dry-run error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "[dry-run] Perfil resuelto: pers") {
		t.Errorf("expected dry-run profile, got: %s", out)
	}
	if !strings.Contains(out, "GIT_SSH_COMMAND=") {
		t.Errorf("expected GIT_SSH_COMMAND in dry-run, got: %s", out)
	}
	if !strings.Contains(out, "git -C mi-app config core.sshCommand") {
		t.Errorf("expected core.sshCommand binding, got: %s", out)
	}
	if !strings.Contains(out, "git -C mi-app config user.name \"Pers Name\"") {
		t.Errorf("expected user.name binding, got: %s", out)
	}
	if !strings.Contains(out, "git -C mi-app config user.email \"pers@mail.com\"") {
		t.Errorf("expected user.email binding, got: %s", out)
	}
}

func TestCloneAndBindingOnDisk(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Crear un bare repo local como origen
	bareRepo := filepath.Join(tmpDir, "origin.git")
	if err := exec.Command("git", "init", "--bare", bareRepo).Run(); err != nil {
		t.Fatalf("git init --bare error: %v", err)
	}

	// 2. Crear repo de trabajo para hacer primer commit y push al bare repo
	workDir := filepath.Join(tmpDir, "initial_work")
	_ = os.MkdirAll(workDir, 0700)
	_ = exec.Command("git", "-C", workDir, "init").Run()
	_ = exec.Command("git", "-C", workDir, "config", "user.name", "Initial").Run()
	_ = exec.Command("git", "-C", workDir, "config", "user.email", "initial@test.com").Run()
	_ = os.WriteFile(filepath.Join(workDir, "README.md"), []byte("# Test"), 0600)
	_ = exec.Command("git", "-C", workDir, "add", ".").Run()
	_ = exec.Command("git", "-C", workDir, "commit", "-m", "init").Run()
	_ = exec.Command("git", "-C", workDir, "push", bareRepo, "HEAD:main").Run()

	// 3. Clave SSH de prueba
	fakeKey := filepath.Join(tmpDir, "id_test_key")
	_ = os.WriteFile(fakeKey, []byte("fake-key"), 0600)

	destDir := filepath.Join(tmpDir, "cloned_repo")
	p := profile.Profile{
		Name:    "trabajo",
		Key:     fakeKey,
		GitName: "Desarrollador Trabajo",
		Email:   "dev@empresa.com",
	}

	// 4. Ejecutar Execute con NoVerify=true hacia el bare repo local
	buf := new(bytes.Buffer)
	err := Execute(CloneOptions{
		Profile:  p,
		URL:      bareRepo,
		Into:     destDir,
		NoVerify: true,
		Stdout:   buf,
	})
	if err != nil {
		t.Fatalf("Execute error: %v\nOutput: %s", err, buf.String())
	}

	// 5. Verificar que el repositorio clonado tenga las configuraciones locales atadas
	outSSH, err := exec.Command("git", "-C", destDir, "config", "--local", "--get", "core.sshCommand").Output()
	if err != nil {
		t.Fatalf("failed to get core.sshCommand: %v", err)
	}
	expectedSSH := "ssh -i " + fakeKey + " -o IdentitiesOnly=yes"
	if strings.TrimSpace(string(outSSH)) != expectedSSH {
		t.Errorf("expected core.sshCommand %q, got %q", expectedSSH, strings.TrimSpace(string(outSSH)))
	}

	outName, err := exec.Command("git", "-C", destDir, "config", "--local", "--get", "user.name").Output()
	if err != nil {
		t.Fatalf("failed to get user.name: %v", err)
	}
	if strings.TrimSpace(string(outName)) != "Desarrollador Trabajo" {
		t.Errorf("expected user.name 'Desarrollador Trabajo', got %q", strings.TrimSpace(string(outName)))
	}

	outEmail, err := exec.Command("git", "-C", destDir, "config", "--local", "--get", "user.email").Output()
	if err != nil {
		t.Fatalf("failed to get user.email: %v", err)
	}
	if strings.TrimSpace(string(outEmail)) != "dev@empresa.com" {
		t.Errorf("expected user.email 'dev@empresa.com', got %q", strings.TrimSpace(string(outEmail)))
	}
}
