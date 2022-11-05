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

func msgSlug(s string) string {
	b := sha256.Sum256([]byte(s))
	return hex.EncodeToString(b[:20])
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
`, env.GetHeader("Subject"), time.Now().Format(`2006-01-02 15:04:05.999999999 -0700 MST`), body


	slug := msgSlug(env.GetHeader("Subject"))
	n := 1
	for _, attachment := range env.Inlines {
		if !strings.HasPrefix(attachment.ContentType, "image/") {
			continue
		}
		file := filepath.ToSlash(filepath.Join("assets", slug+fmt.Sprintf("-%03d.jpg", n)))
		err = saveJpeg(file, attachment)
		if err != nil {
			log.Fatalf("cannot write attachment file: %v", err)
		}
		err = exec.Command("git", "add", file).Run()
		if err != nil {
			log.Fatalf("cannot execute git add: %v", err)
		}

		marker := "[image: " + attachment.FileName + "]"
		text = strings.ReplaceAll(text, marker, fmt.Sprintf(`![%s](%s)`, attachment.FileName, "/"+file))
		n++
	}

	for _, attachment := range env.OtherParts {
		if !strings.HasPrefix(attachment.ContentType, "image/") {
			continue
		}
		file := filepath.ToSlash(filepath.Join("assets", slug+fmt.Sprintf("-%03d.jpg", n)))
		err = saveJpeg(file, attachment)
		if err != nil {
			log.Fatalf("cannot write attachment file: %v", err)
		}
		err = exec.Command("git", "add", file).Run()
		if err != nil {
			log.Fatalf("cannot execute git add: %v", err)
		}

		text += fmt.Sprintf("![%s](%s)\n\n", attachment.FileName, "/"+file)
		n++
	}

	file := filepath.Join("_posts", time.Now().Format("2006-01-02")+"-"+slug+".md")
	err = ioutil.WriteFile(file, []byte(text), 0644)
	if err != nil {
		log.Fatalf("cannot create new entry: %v", err)
	}
	err = exec.Command("git", "add", file).Run()
	if err != nil {
		log.Fatalf("cannot execute git add: %v", err)
	}
	err = exec.Command("git", "commit", "--no-gpg-sign", "-a", "-m", "Add entry: "+env.GetHeader("Subject")).Run()
	if err != nil {
		log.Fatalf("cannot execute git commit: %v", err)
	}
	err = exec.Command("git", "push", "--force", "origin", "master").Run()
	if err != nil {
		log.Fatalf("cannot execute git push: %v", err)
	}

	from := mail.Address{Name: "moblog", Address: sender}
	message := fmt.Sprintf(`To: %s
From: %s
Reference: %s
Subject: %s
投稿が完了しました
`, addr.String(), from.String(), env.GetHeader("Message-ID"), "RE: "+env.GetHeader("Subject"))

	err = smtp.SendMail(mailserver, nil, from.Address, []string{addr.Address}, []byte(message))
	if err != nil {
		log.Fatalf("cannot send e-mail: %v", err)
	}
}