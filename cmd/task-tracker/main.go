// Command task-cli is a small CLI for managing tasks stored in a JSON file.
//
// Usage:
//
//	task-cli add -d "write the report" [-s todo|"in progress"|done]
//	task-cli list [-s todo|"in progress"|done]
//	task-cli get <id>
//	task-cli update <id> -d "new text"
//	task-cli mark-done <id>
//	task-cli mark-in-progress <id>
//	task-cli delete <id>
//
// Tasks are persisted to tasks.json in the current working directory.
package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"task-tracker/internal/store"
	"task-tracker/internal/task"
)

// program is the name shown in every help and usage message.
const program = "task-cli"

// storeFile is the JSON file the CLI reads and writes, relative to the working
// directory.
const storeFile = "tasks.json"

// displayTime is how timestamps are rendered in the terminal.
const displayTime = "2006-01-02 15:04:05"

var usage = program + ` - manage simple tasks in ` + storeFile + `

Usage:
  ` + program + ` add -d <description> [-s <status>]   add a task (status defaults to "todo")
  ` + program + ` list [-s <status>]                   list tasks, optionally filtered by status
  ` + program + ` update <id> -d <description>         change a task's description
  ` + program + ` mark-done <id>                       set a task's status to "done"
  ` + program + ` mark-in-progress <id>                set a task's status to "in progress"
  ` + program + ` delete <id>                          remove a task
  ` + program + ` help                                 show this message

Statuses: ` + task.StatusList() + `
`

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}

	cmd, rest := args[0], args[1:]

	if cmd == "help" || cmd == "-h" || cmd == "--help" {
		fmt.Print(usage)
		return
	}

	opts := parse(rest)
	if opts.help {
		fmt.Print(usage)
		return
	}

	var err error
	switch cmd {
	case "add":
		err = cmdAdd(opts)
	case "list", "ls":
		err = cmdList(opts)
	case "update":
		err = cmdUpdate(opts)
	case "mark-done":
		err = cmdSetStatus(opts, task.StatusDone)
	case "mark-in-progress":
		err = cmdSetStatus(opts, task.StatusInProgress)
	case "delete", "rm":
		err = cmdDelete(opts)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", cmd, usage)
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func cmdAdd(opts options) error {
	description := strings.TrimSpace(opts.description)
	if description == "" {
		return errors.New(`missing description (use -d "...")`)
	}

	status := task.StatusTodo
	if opts.hasStatus {
		st, err := task.ParseStatus(opts.status)
		if err != nil {
			return err
		}
		status = st
	}

	s, err := openStore()
	if err != nil {
		return err
	}

	t, err := s.Add(description, status)
	if err != nil {
		return err
	}
	fmt.Printf("added task %d\n", t.ID)
	printTask(t)
	return nil
}

func cmdList(opts options) error {
	var filter *task.Status
	if opts.hasStatus {
		st, err := task.ParseStatus(opts.status)
		if err != nil {
			return err
		}
		filter = &st
	}

	s, err := openStore()
	if err != nil {
		return err
	}

	tasks := s.List(filter)
	if len(tasks) == 0 {
		fmt.Println("no tasks found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSTATUS\tDESCRIPTION\tCREATED\tUPDATED")
	for _, t := range tasks {
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n",
			t.ID, t.Status, t.Description,
			t.CreatedAt.Local().Format(displayTime),
			t.UpdatedAt.Local().Format(displayTime))
	}
	return w.Flush()
}

// cmdUpdate changes the description of a task. Status changes live in the
// mark-done and mark-in-progress commands.
func cmdUpdate(opts options) error {
	id, err := opts.taskID()
	if err != nil {
		return err
	}

	description := strings.TrimSpace(opts.description)
	if description == "" {
		return errors.New(`missing new description (use -d "...")`)
	}

	s, err := openStore()
	if err != nil {
		return err
	}

	t, err := s.Update(id, &description, nil)
	if err != nil {
		return err
	}
	fmt.Printf("updated task %d\n", t.ID)
	printTask(t)
	return nil
}

// cmdSetStatus implements the mark-done and mark-in-progress commands.
func cmdSetStatus(opts options, status task.Status) error {
	id, err := opts.taskID()
	if err != nil {
		return err
	}

	s, err := openStore()
	if err != nil {
		return err
	}

	t, err := s.Update(id, nil, &status)
	if err != nil {
		return err
	}
	fmt.Printf("marked task %d as %s\n", t.ID, t.Status)
	printTask(t)
	return nil
}

func cmdDelete(opts options) error {
	id, err := opts.taskID()
	if err != nil {
		return err
	}

	s, err := openStore()
	if err != nil {
		return err
	}

	t, err := s.Delete(id)
	if err != nil {
		return err
	}
	fmt.Printf("deleted task %d (%s)\n", t.ID, t.Description)
	return nil
}

// openStore returns a store loaded from storeFile.
func openStore() (*store.Store, error) {
	s := store.NewStore(storeFile)
	if err := s.Load(); err != nil {
		return nil, err
	}
	return s, nil
}

// options holds the values parsed from a command's arguments.
type options struct {
	idArg       string // first bare argument, when present
	hasID       bool
	description string
	status      string
	hasStatus   bool
	help        bool
}

// taskID converts the positional argument into a task id.
func (o options) taskID() (int, error) {
	if !o.hasID {
		return 0, fmt.Errorf("missing task id (usage: %s <command> <id>)", program)
	}
	id, err := strconv.Atoi(o.idArg)
	if err != nil {
		return 0, fmt.Errorf("invalid task id %q: must be an integer", o.idArg)
	}
	return id, nil
}

// parse extracts the flags this CLI understands from argv. Flags it does not
// know are ignored rather than reported, so a stray or obsolete flag never
// fails a command. A known flag takes its value either separately ("-d text")
// or inline ("-d=text"), and the first bare argument is the task id.
func parse(argv []string) options {
	var opts options

	for i := 0; i < len(argv); i++ {
		arg := argv[i]

		if !strings.HasPrefix(arg, "-") || arg == "-" {
			if !opts.hasID {
				opts.idArg, opts.hasID = arg, true
			}
			continue
		}

		name, value, inline := splitFlag(arg)
		switch name {
		case "h", "help":
			opts.help = true
		case "d", "description":
			if !inline {
				value, i = followingValue(argv, i)
			}
			opts.description = value
		case "s", "status":
			if !inline {
				value, i = followingValue(argv, i)
			}
			opts.status, opts.hasStatus = value, true
		default:
			// Unrecognised flag: ignored on purpose.
		}
	}

	return opts
}

// splitFlag breaks "-d=text" and "--description=text" into the flag name
// without leading dashes, its inline value, and whether a value was inline.
func splitFlag(arg string) (name, value string, inline bool) {
	name = strings.TrimLeft(arg, "-")
	if n, v, ok := strings.Cut(name, "="); ok {
		return n, v, true
	}
	return name, "", false
}

// followingValue returns the token after a flag as its value, advancing i past
// it. When the flag ends the command line the value is empty.
func followingValue(argv []string, i int) (string, int) {
	if i+1 < len(argv) {
		return argv[i+1], i + 1
	}
	return "", i
}

// printTask renders a single task in a labelled block.
func printTask(t task.Task) {
	fmt.Printf("id:          %d\n", t.ID)
	fmt.Printf("description: %s\n", t.Description)
	fmt.Printf("status:      %s\n", t.Status)
	fmt.Printf("created at:  %s\n", t.CreatedAt.Local().Format(displayTime))
	fmt.Printf("updated at:  %s\n", t.UpdatedAt.Local().Format(displayTime))
}
