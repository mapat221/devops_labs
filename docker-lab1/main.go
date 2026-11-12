package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

type Process struct {
	Args []string `json:"args"`
	Cwd  string   `json:"cwd"`
}

type Root struct {
	Path string `json:"path"`
}

type Namespace struct {
	Type string `json:"type"`
}

type Linux struct {
	Namespaces []Namespace `json:"namespaces"`
}

type Config struct {
	OciVersion string  `json:"ociVersion"`
	ID         string  `json:"id"`
	Hostname   string  `json:"hostname"`
	Process    Process `json:"process"`
	Root       Root    `json:"root"`
	Linux      Linux   `json:"linux"`
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "child" {
		runContainer()
		return
	}

	file, err := os.ReadFile("config.json")
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	var config Config

	err = json.Unmarshal(file, &config)
	if err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	baseDir := filepath.Join("/var/lib", "docker-lab", config.ID)
	upperDir := filepath.Join(baseDir, "upper")
	workDir := filepath.Join(baseDir, "work")
	mergedDir := filepath.Join(baseDir, "merged")

	err = os.MkdirAll(baseDir, 0755)
	if err != nil {
		fmt.Printf("eror creating dir %s: %v\n", baseDir, err)
		os.Exit(1)
	}

	err = os.MkdirAll(upperDir, 0755)
	if err != nil {
		fmt.Printf("eror creating dir %s: %v\n", upperDir, err)
		os.Exit(1)
	}

	err = os.MkdirAll(workDir, 0755)
	if err != nil {
		fmt.Printf("eror creating dir %s: %v\n", workDir, err)
		os.Exit(1)
	}

	err = os.MkdirAll(mergedDir, 0755)
	if err != nil {
		fmt.Printf("eror creating dir %s: %v\n", mergedDir, err)
		os.Exit(1)
	}

	cmd := exec.Command("/proc/self/exe", "child")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWUTS,
	}

	if err := cmd.Run(); err != nil {
		fmt.Printf("eror executing: %v\n", err)
		os.Exit(1)
	}
}

func runContainer() {
	file, _ := os.ReadFile("config.json")
	var config Config
	json.Unmarshal(file, &config)

	baseDir := filepath.Join("/var/lib", "docker-lab", config.ID)
	upperDir := filepath.Join(baseDir, "upper")
	workDir := filepath.Join(baseDir, "work")
	mergedDir := filepath.Join(baseDir, "merged")
	lowerDir, _ := filepath.Abs(config.Root.Path)
	syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, "")
	syscall.Sethostname([]byte(config.Hostname))
	mountOptions := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", lowerDir, upperDir, workDir)
	syscall.Mount("overlay", mergedDir, "overlay", 0, mountOptions)
	syscall.Chroot(mergedDir)
	os.Chdir(config.Process.Cwd)
	syscall.Exec(config.Process.Args[0], config.Process.Args, os.Environ())
}
