package commandexec

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Avoid cmd.exe expansion entirely: execute npm's JavaScript entry point with Node.
func resolveCommand(name string, args []string) (string, []string, error) {
	base := strings.ToLower(filepath.Base(name))
	if base == "npm" || base == "npm.cmd" {
		npm, err := exec.LookPath(name)
		if err != nil {
			return "", nil, err
		}
		script := filepath.Join(filepath.Dir(npm), "node_modules", "npm", "bin", "npm-cli.js")
		if _, err := os.Stat(script); err != nil {
			return "", nil, fmt.Errorf("locate npm CLI beside %s: %w", npm, err)
		}
		node := filepath.Join(filepath.Dir(npm), "node.exe")
		if _, err := os.Stat(node); err != nil {
			node, err = exec.LookPath("node.exe")
			if err != nil {
				return "", nil, err
			}
		}
		return node, append([]string{script}, args...), nil
	}
	ext := strings.ToLower(filepath.Ext(name))
	if ext == ".cmd" || ext == ".bat" {
		return "", nil, fmt.Errorf("batch tool %q is unsupported; configure a native executable", name)
	}
	return name, args, nil
}

func configure(cmd *exec.Cmd) {
	// Assign the suspended child to a kill-on-close job before it can spawn children.
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_SUSPENDED}
}

func Run(cmd *exec.Cmd) error {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(job)
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	process, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE|windows.PROCESS_SUSPEND_RESUME, false, uint32(cmd.Process.Pid))
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return err
	}
	defer windows.CloseHandle(process)
	if err := windows.AssignProcessToJobObject(job, process); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return err
	}
	// exec does not expose the initial thread handle. Resume the suspended process.
	status, _, _ := windows.NewLazySystemDLL("ntdll.dll").NewProc("NtResumeProcess").Call(uintptr(process))
	if status != 0 {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return fmt.Errorf("resume child process: NTSTATUS %#x", status)
	}
	// CommandContext terminates the direct child; closing the job after Wait also
	// terminates every descendant. WaitDelay bounds inherited output pipes.
	return cmd.Wait()
}
