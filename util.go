package main

import (
	"strings"
	"fmt"
	"os"
)

// encode 
func encodeUTF(s string) string {
	// TODO check to implement a real encoding
	s = strings.Replace(s, "!", "U+0021", -1)
	s = strings.Replace(s, "\"", "U+0022", -1)
	s = strings.Replace(s, "#", "U+0023", -1)
	s = strings.Replace(s, "&", "U+0024", -1)
	s = strings.Replace(s, "%", "U+0025", -1)
	s = strings.Replace(s, "&", "U+0026", -1)
	s = strings.Replace(s, "'", "U+0027", -1)
	s = strings.Replace(s, "(", "U+0028", -1)
	s = strings.Replace(s, ")", "U+0029", -1)
	s = strings.Replace(s, "*", "U+002A", -1)
	s = strings.Replace(s, "+", "U+002B", -1)
	s = strings.Replace(s, ",", "U+002C", -1)
	s = strings.Replace(s, "-", "U+002D", -1)
	s = strings.Replace(s, "/", "U+002F", -1)
	return s
}

func checkErr(err error) {
	if err != nil {
		fmt.Print(err)
		panic(err)
		os.Exit(1)
	}
}

func getMySqlSchema(value string) string {
	s := strings.Split(value, "/")
	desiredSchema := s[len(s)-1]
	return desiredSchema
}



func eqstring(s1 string, s2 string) (v bool) {
	if s1 == s2 {
		v = true
		return
	}
	v = false
	return
}

