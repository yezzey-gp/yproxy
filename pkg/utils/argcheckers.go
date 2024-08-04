package utils

import "fmt"

func countCheck(args []string, count int64) error {
	if len(args) != int(count) {
		return fmt.Errorf("invalid number of arguments, %d expected, %d received", count, len(args))
	} else {
		return nil
	}
}
func lessThan(args []string, count int64) error {
	if len(args) > int(count) {
		return fmt.Errorf("invalid number of arguments, %d<= expected, %d received", count, len(args))
	} else {
		return nil
	}
}

func ZeroArg(args []string) error {
	return countCheck(args, 0)
}

func OneArg(args []string) error {
	return countCheck(args, 1)
}
