package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net/mail"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/jhillyerd/enmime"
	"github.com/mattn/godown"
)

func clean() error {
	commands := [][]string{
		{"git", "clean"},
		{"git", "checkout", "."},
		{"git", "reset", "--hard", "HEAD"},
		{"git", "clean", "-fdx"},
		// --force, diectory(reverse),
		// -x:Don’t use the standard ignore rules
	}

	for _, args := range commands {
		err := exec.Command(args[0], args[1:]...).Run()
		if err != nil {
			return err
		}
	}
	return nil
}

func main() {
	var mailserver string
	var accept string
	var sender string
	var repo string
	var usehtml bool
	flag.StringVar(&mailserver, "m", "localhost:25", "Mail Server")
	flag.StringVar(&accept, "a", "*", "Accept E-mail From")
	flag.StringVar(&sender, "s", "moblog@example.com", "E-mail Sender")
	flag.StringVar(&repo, "d", "/path/to/jekyll/blog", "repository of jekyll")
	flag.BoolVar(&usehtml, "t", false, "Use HTML")
	flag.Parse()

	// Change Working Directory
	err := os.Chdir(repo)
	if err != nil {
		log.Fatalf("Cannot Chdir: %v", err)
	}

	// TODO
	err = clean()
	if err != nil {
		log.Fatalf("Cannot Git Clean: %v", err)
	}

	env, error := enmime.ReadEnvelope(os.Stdin)
	if error != nil {
		log.Fatalf("Cannot Parse E-mail: %v", error)
	}

	addr, error := mail.ParseAddress(env.GetHeader("From"))
	if error != nil {
		log.Fatalf("Cannot Parse Address: %v", error)
	}

	if usehtml && env.HTML != "" {
		var buf bytes.Buffer
		if err := godown.Convert(&buf, strings.NewReader(env.HTML), nil); err == nil {
			env.Text = buf.String()
		}
	}

	body := strings.ReplaceAll(strings.ReplaceAll(env.Text, "\r", ""), "\n", "\n\n")
	text := fmt.Sprintf(`---
layout: post
title: %s
date: %s
---
%s
`, env.GetHeader("Subject"), time.Now().Format(`2006-01-02 15:04:05.999999999 -0700 MST`), body)

}
