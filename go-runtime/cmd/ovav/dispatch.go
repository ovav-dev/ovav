// dispatch.go — Testable command routing extracted from main.go
//
// The routeCommand function replaces the monolithic switch in main(),
// returning an exit code instead of calling os.Exit directly.
// This makes the routing logic unit-testable.

package main

import (
	"fmt"
	"os"
)

// routeCommand dispatches a CLI command and returns an exit code.
// It does NOT call os.Exit — the caller (main) is responsible for that.
// This separation enables unit testing of the routing logic.
func routeCommand(cmd string, args []string) int {
	switch cmd {
	case "status":
		return cmdStatus(args)
	case "profile":
		return cmdProfile(args)
	case "config":
		return cmdConfig(args)
	case "tools":
		return cmdTools(args)
	case "doctor", "health":
		return cmdDoctor(args)
	case "update":
		return cmdUpdate(args)
	case "vault":
		return cmdVault(args)
	case "tailor":
		return cmdTailor(args)
	case "waiver":
		return cmdWaiver(args)
	case "ceo":
		return cmdCeo(args)
	case "version", "--version", "-v":
		return cmdVersion(args)
	case "install":
		return cmdInstall(args)
	case "uninstall":
		return cmdUninstall(args)
	case "plan":
		return cmdPlan(args)
	case "backup":
		return cmdBackup(args)
	case "apply":
		return cmdApply(args)
	case "verify":
		return cmdVerify(args)
	case "restore", "rollback":
		return cmdRestore(args)
	case "deploy":
		return cmdDeploy(args)
	case "sbom":
		return cmdSBOM(args)
	case "cockpit", "shell", "tui":
		return cmdCockpit(args)
	case "project":
		return cmdProject(args)
	case "worktree", "wt":
		return cmdWorktree(args)
	case "own", "nuke":
		return cmdOwn(args)
	case "chronos":
		return cmdChronos(args)
	case "hook":
		return cmdHook(args)
	case "infra":
		return cmdInfra(args)
	case "login", "signin", "auth":
		return cmdLogin(args)
	case "whoami", "identity":
		return cmdWhoami(args)
	case "logout", "signout":
		return cmdLogout(args)
	case "license":
		return cmdLicense(args)
	case "govern":
		return cmdGovern(args)
	case "product":
		return cmdProduct(args)
	case "defend":
		return cmdDefend(args)
	case "surfaces":
		return cmdSurfaces(args)
	case "export-gate", "publish-check":
		return cmdExportGate(args)
	case "repo-check", "presentation-check":
		return cmdRepoCheck(args)
	case "release-check", "rc-check":
		return cmdReleaseCheck(args)
	case "fresh-smoke", "dogfood":
		return cmdFreshSmoke(args)
	case "detect-env":
		return cmdDetectEnv(args)
	case "gateway":
		return cmdGateway(args)
	case "sync":
		return cmdSync(args)
	case "convert":
		return cmdConvert(args)
	case "resolve-subagent", "resolve_subagent":
		return cmdResolveSubagent(args)
	case "delegate":
		return cmdDelegate(args)
	case "validate":
		return cmdValidate(args)
	case "push":
		return cmdPush(args)
	case "memory", "mem":
		return cmdMemory(args)
	case "help", "--help", "-h":
		printUsage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\nRun 'ovav help' for usage.\n", cmd)
		return 2
	}
}

// knownCommands returns the set of all recognized top-level command strings.
// Useful for testing and introspection.
func knownCommands() []string {
	return []string{
		"status", "profile", "config", "tools", "doctor", "health",
		"update", "vault", "tailor", "waiver", "ceo",
		"version", "--version", "-v",
		"install", "uninstall", "plan", "backup", "apply", "verify",
		"restore", "rollback", "deploy", "sbom",
		"cockpit", "shell", "tui",
		"project", "git", "worktree", "wt",
		"chronos", "hook", "infra",
		"login", "signin", "auth",
		"whoami", "identity",
		"logout", "signout",
		"license", "govern", "defend", "product", "surfaces",
		"export-gate", "publish-check",
		"repo-check", "presentation-check",
		"release-check", "rc-check",
		"fresh-smoke", "dogfood",
		"detect-env", "gateway",
		"sync",
		"resolve-subagent", "resolve_subagent",
		"delegate",
		"validate",
		"push",
		"memory", "mem",
		"help", "--help", "-h",
	}
}

// isKnownCommand checks if a command string is recognized by the router.
func isKnownCommand(cmd string) bool {
	for _, k := range knownCommands() {
		if k == cmd {
			return true
		}
	}
	return false
}
