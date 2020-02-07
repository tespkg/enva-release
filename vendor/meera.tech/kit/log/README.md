# log

A log facility based on uber zap log, which is work with cobra command closely and trying to replace logrus usage in our code.

## Example

```go
package cmd

import (
	"github.com/spf13/cobra"
	"meera.tech/kit/log"
)

func serveCmd() *cobra.Command {
    loggingOptions := log.DefaultOptions()

	cmd := &cobra.Command{
		Use:   "serve",
		RunE: func(cmd *cobra.Command, args []string) error{
            loggingOptions.SetLogCallers("default", true)
            if err := log.Configure(loggingOptions); err != nil {
                return err
            }
            return nil
		},
	}

    // Attach all these flags by add only one line.
    // --log_as_json                   Whether to format output as JSON or in plain console-friendly format
    // --log_caller string             Comma-separated list of scopes for which to include called information, scopes can be any of [default]
    // --log_output_level string       The minimum logging level of messages to output,  can be one of [debug, info, warn, error, fatal, none] (default "default:info")
    // --log_rotate string             The path for the optional rotating log file
    // --log_rotate_max_age int        The maximum age in days of a log file beyond which the file is rotated (0 indicates no limit) (default 30)
    // --log_rotate_max_backups int    The maximum number of log file backups to keep before older files are deleted (0 indicates no limit) (default 1000)
    // --log_rotate_max_size int       The maximum size in megabytes of a log file beyond which the file is rotated (default 104857600)
    // --log_stacktrace_level string   The minimum logging level at which stack traces are captured, can be one of [debug, info, warn, error, fatal, none] (default "default:none")
    // --log_target stringArray        The set of paths where to output the log. This can be any path as well as the special values stdout and stderr (default [stdout])
    loggingOptions.AttachCobraFlags(cmd)

    return cmd
}

func somefunc() {
    log.Infoa("hello world!")
}
```