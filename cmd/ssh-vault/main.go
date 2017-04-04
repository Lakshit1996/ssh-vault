package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"

	"github.com/ssh-vault/crypto"
	"github.com/ssh-vault/crypto/aead"
	sv "github.com/ssh-vault/ssh-vault"
)

var version string

func exit1(err error) {
	fmt.Println(err)
	os.Exit(1)
}

func main() {
	var (
		f       = flag.Bool("f", false, "Print ssh key `fingerprint`")
		k       = flag.String("k", "~/.ssh/id_rsa.pub", "Public `ssh key or index` when using option -u")
		o       = flag.String("o", "", "Write output to `file` instead of stdout. Only for option view")
		u       = flag.String("u", "", "GitHub `username or URL`, optional [-k N] where N is the key index to use")
		v       = flag.Bool("v", false, fmt.Sprintf("Print version: %s", version))
		options = []string{"create", "edit", "view"}
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [-f fingerprint] [-k key] [-o file] [-u user] [create|edit|view] vault\n\n%s\n%s\n%s\n%s\n%s\n%s\n%s\n\n",
			os.Args[0],
			"  Options:",
			"    create    Creates a new vault, if no vault defined outputs to stdout.",
			"              Can read from stdin, example:",
			"                  echo \"secret\" | ssh-vault -u <user> create",
			"    edit      Edit an existing vault.",
			"    view      View an existing vault, can read from stdin, example:",
			"                  echo \"SSH-VAULT...\" | ssh-vault view",
		)
		flag.PrintDefaults()
	}

	flag.Parse()

	if *v {
		fmt.Printf("%s\n", version)
		os.Exit(0)
	}

	if flag.NArg() < 1 && !*f {
		exit1(fmt.Errorf("Missing option, use (\"%s -h\") for help.\n", os.Args[0]))
	}

	usr, _ := user.Current()
	if len(*k) > 2 {
		if (*k)[:2] == "~/" {
			*k = filepath.Join(usr.HomeDir, (*k)[2:])
		}
	}

	vault, err := sv.New(*k, *u, flag.Arg(0), flag.Arg(1))
	if err != nil {
		exit1(err)
	}

	// ssh-keygen -f id_rsa.pub -e -m PKCS8
	if err := vault.PKCS8(); err != nil {
		exit1(err)
	}

	// print fingerprint and exit
	if *f {
		fmt.Println(vault.Fingerprint)
		os.Exit(0)
	}

	// check options
	exit := true
	for _, v := range options {
		if flag.Arg(0) == v {
			exit = false
			break
		}
	}
	if exit {
		exit1(fmt.Errorf("Invalid option, use (\"%s -h\") for help.\n", os.Args[0]))
	}

	vault.Password, err = crypto.GenerateNonce(32)
	if err != nil {
		exit1(err)
	}

	switch flag.Arg(0) {
	case "create":
		data, err := vault.Create()
		if err != nil {
			exit1(err)
		}
		out, err := aead.Encrypt(vault.Password, data, []byte(vault.Fingerprint))
		if err != nil {
			exit1(err)
		}
		err = vault.Close(out)
		if err != nil {
			exit1(err)
		}
	case "edit":
		data, err := vault.View()
		if err != nil {
			exit1(fmt.Errorf("Missing vault name, use (\"%s -h\") for help.\n", os.Args[0]))
		}
		out, err := vault.Edit(data)
		if err != nil {
			exit1(err)
		}
		out, err = aead.Encrypt(vault.Password, out, []byte(vault.Fingerprint))
		if err != nil {
			exit1(err)
		}
		err = vault.Close(out)
		if err != nil {
			exit1(err)
		}
	case "view":
		out, err := vault.View()
		if err != nil {
			exit1(err)
		}
		if *o != "" {
			if err := ioutil.WriteFile(*o, out, 0600); err != nil {
				exit1(err)
			}
		} else {
			fmt.Printf("%s", out)
		}
	}
}
