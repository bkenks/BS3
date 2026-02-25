package injector

import (
	"fmt"
	"os"
	"os/exec"
)

// =====================================================
// Injector
// =====================================================
// Run sets environment variables from the provided secrets map
// and optionally executes a command with those variables.
//
// If 'command' is empty, it just prints the secrets to stdout.
// Otherwise, it runs the given command with the secrets injected.

// ~~~ `injector` Package Entrypoint ~~~
func Run(secrets map[string]string, command []string) error {
	// ~~~ Current Environment ~~~
	// get current environment variables
	env := os.Environ()

	// ~~~ Env Var Assignment ~~~
	// Set each k,v pair as an environment variable for current shell session
	for k, v := range secrets {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// ~~~ Prepare Commands ~~~
	// command[0] is the executable (e.g. `docker`)
	// command[1:] are the arguements (e.g. `compose up`)
	cmd := exec.Command(command[0], command[1:]...)

	// ~~~ Set Environment For Command ~~~
	cmd.Env = env

	// ~~~ Process Connection ~~~
	// connect command (child) process's input/output to current process
	cmd.Stdout = os.Stdout // forwards standard output
	cmd.Stderr = os.Stderr // forwards error output
	cmd.Stdin = os.Stdin   // forwards input if needed

	// ~~~ Run Command ~~~
	return cmd.Run()
}

// =====================================================
// END "Injector"
// =====================================================
