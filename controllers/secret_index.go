package controllers

import "fmt"

func secretIndexValue(namespace, name string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}
