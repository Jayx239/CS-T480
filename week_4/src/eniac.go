package main

import (
	"fmt"
	"bufio"
	"strings"
	"strconv"
	"os"
	"runtime"
	"time"
	"unicode"
	"flag"
)

type pulse struct {
	val int
	resp chan int
}

var initbut, cycbut chan int
var ppunch chan string
var mpsw, conssw, cycsw, multsw, divsw, prsw chan [2]string
var accsw [20]chan [2]string
var ftsw [3]chan [2]string
var width, height int
var cardscanner *bufio.Scanner
var punchwriter *bufio.Writer

func b2is(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func fanout(in chan pulse, out []chan pulse) {
	var q pulse

	q.resp = make(chan int)
	for {
		p :=<- in
		nresp := 0
		if p.val != 0 {
			q.val = p.val
			for _,c := range out {
foo:
				for {
					select {
					case c <- q:
						break foo
					case <- q.resp:
						nresp++
					}
				}
			}
		}
		for nresp < len(out) {
			<- q.resp
			nresp++
		}
		p.resp <- 1
	}
}

func proccmd(cmd string) int {
	f := strings.Fields(cmd)
	if len(f) == 0 {
		return 0
	}
	switch f[0] {
	case "#":		// Just a comment.  Ignore the line
		break
	case "b":
		if len(f) != 2 {
			fmt.Println("button syntax: b button")
			break
		}
		switch f[1] {
		case "c":
			initbut <- 5
		case "i":
			initbut <- 4
		case "p":
			cycbut <- 1
		case "r":
			initbut <- 3
		}
	case "d":
		if len(f) != 2 {
			fmt.Println("Status syntax: d unit")
			break
		}
		p := strings.Split(f[1], ".")
		switch p[0] {
		case "a":
			if len(p) != 2 {
				fmt.Println("Accumulator print syntax: d a.unit")
			} else {
				unit, _ := strconv.Atoi(p[1])
				fmt.Println(accstat(unit-1))
			}
		case "c":
			fmt.Println(consstat())
		case "d":
			fmt.Println(divsrstat2())
		case "f":
			if len(p) != 2 {
				fmt.Println("Function table print syntax: d f.unit")
			} else {
				unit, _ := strconv.Atoi(p[1])
				fmt.Println(ftstat(unit-1))
			}
		case "i":
			fmt.Println(initstat())
		case "m":
			fmt.Println(multstat())
		case "p":
			fmt.Println(mpstat())
		}
	case "D":
		fmt.Println(initstat())
		fmt.Println(mpstat())
		for i := 0; i < 20; i += 2 {
			fmt.Print(accstat(i))
			fmt.Print("   ")
			fmt.Println(accstat(i+1))
		}
		fmt.Println(divsrstat2())
		fmt.Println(multstat())
		for i := 0; i < 3; i++ {
			fmt.Println(ftstat(i))
		}
		fmt.Println(consstat())
	case "f":
		if len(f) != 3 {
			fmt.Println("file syntax: f (r|p) filename")
			break
		}
		switch f[1] {
		case "r":
			fp, err := os.Open(f[2])
			if err != nil {
				fmt.Printf("Card reader open: %s\n", err)
				break
			}
			cardscanner = bufio.NewScanner(fp)
		case "p":
			fp, err := os.Create(f[2])
			if err != nil {
				fmt.Printf("Card punch open: %s\n", err)
				break
			}
			punchwriter = bufio.NewWriter(fp)
		}
	case "l":
		if len(f) != 2 {
			fmt.Println("Load syntax: l file")
			break
		}
		fd, err := os.Open("programs/" + f[1])
		if err != nil {
			fmt.Println(err)
			break
		}
		sc := bufio.NewScanner(fd)
		for sc.Scan() {
			if proccmd(sc.Text()) < 0 {
				break
			}
		}
		fd.Close()
	case "p":
		if len(f) != 3 {
			fmt.Println("Invalid jumper spec", cmd)
			break
		}
		p1 := strings.Split(f[1], ".")
		p2 := strings.Split(f[2], ".")
		ch := make(chan pulse)
		switch {
		case p1[0] == "ad":
			if len(p1) != 4 {
				fmt.Println("Adapter jumper syntax: ad.ilk.unit.param")
				break;
			}
			unit, _ := strconv.Atoi(p1[2])
			param, _ := strconv.Atoi(p1[3])
			adplug(p1[1], 1, unit - 1, param, ch)
		case p1[0][0] == 'a':
			if len(p1) != 2 {
				fmt.Println("Accumulator jumper syntax: aunit.terminal")
				break
			}
			unit, _ := strconv.Atoi(p1[0][1:])
			accplug(unit - 1, p1[1], ch)
		case p1[0] == "c":
			if len(p1) != 2 {
				fmt.Println("Invalid constant jumper:", cmd)
				break
			}
			consplug(p1[1], ch)
		case p1[0] == "d":
			if len(p1) != 2 {
				fmt.Println("Divider jumper syntax: d.terminal")
				break
			}
			divsrplug(p1[1], ch)
		case p1[0][0] == 'f':
			if len(p1) != 2 {
				fmt.Println("Function table jumper syntax: funit.terminal")
				break
			}
			unit, _ := strconv.Atoi(p1[0][1:])
			ftplug(unit - 1, p1[1], ch)
		case p1[0] == "i":
			if len(p1) != 2 {
				fmt.Println("Initiator jumper syntax: i.terminal")
				break
			}
			initplug(p1[1], ch)
		case p1[0] == "m":
			if len(p1) != 2 {
				fmt.Println("Multiplier jumper syntax: m.terminal")
				break
			}
			multplug(p1[1], ch)
		case p1[0] == "p":
			mpplug(p1[1], ch)
		case unicode.IsDigit(rune(p1[0][0])):
			hpos := strings.IndexByte(p1[0], '-')
			if hpos == -1 {
				tray, _ := strconv.Atoi(p1[0])
				if tray < 1 {
					fmt.Println("Invalid data trunk", p1[0])
					break
				}
				trunkrecv(0, tray - 1, ch)
			} else {
				tray, _ := strconv.Atoi(p1[0][:hpos])
				line, _ := strconv.Atoi(p1[0][hpos+1:])
				trunkrecv(1, (tray - 1) * 11 + line - 1, ch)
			}
		default:
			fmt.Println("Invalid jack spec: ", p1)
		}
		switch {
		case p2[0] == "ad":
			if len(p2) != 4 {
				fmt.Println("Adapter jumper syntax: ad.ilk.unit.param")
				break;
			}
			unit, _ := strconv.Atoi(p2[2])
			param, _ := strconv.Atoi(p2[3])
			adplug(p2[1], 0, unit - 1, param, ch)
		case p2[0][0] == 'a':
			if len(p2) != 2 {
				fmt.Println("Accumulator jumper syntax: aunit.terminal")
				break
			}
			unit, _ := strconv.Atoi(p2[0][1:])
			accplug(unit - 1, p2[1], ch)
		case p2[0] == "c":
			if len(p2) != 2 {
				fmt.Println("Invalid constant jumper:", cmd)
				break
			}
			consplug(p2[1], ch)
		case p2[0] == "d":
			if len(p2) != 2 {
				fmt.Println("Divider jumper syntax: d.terminal")
				break
			}
			divsrplug(p2[1], ch)
		case p2[0][0] == 'f':
			if len(p2) != 2 {
				fmt.Println("Function table jumper syntax: funit.terminal")
				break
			}
			unit, _ := strconv.Atoi(p2[0][1:])
			ftplug(unit - 1, p2[1], ch)
		case p2[0] == "i":
			if len(p2) != 2 {
				fmt.Println("Initiator jumper syntax: i.terminal")
				break
			}
			initplug(p2[1], ch)
		case p2[0] == "m":
			if len(p2) != 2 {
				fmt.Println("Multiplier jumper syntax: m.terminal")
				break
			}
			multplug(p2[1], ch)
		case p2[0] == "p":
			mpplug(p2[1], ch)
		case unicode.IsDigit(rune(p2[0][0])):
			hpos := strings.IndexByte(p2[0], '-')
			if hpos == -1 {
				tray, _ := strconv.Atoi(p2[0])
				if tray < 1 {
					fmt.Println("Invalid data trunk", p2[0])
					break
				}
				trunkxmit(0, tray - 1, ch)
			} else {
				tray, _ := strconv.Atoi(p2[0][:hpos])
				line, _ := strconv.Atoi(p2[0][hpos+1:])
				trunkxmit(1, (tray - 1) * 11 + line - 1, ch)
			}
		default:
			fmt.Println("Invalid jack spec: ", p2)
		}
	case "q":
		return -1
	case "r":
		if len(f) != 2 {
			fmt.Println("Status syntax: r unit")
			break
		}
		p := strings.Split(f[1], ".")
		switch p[0] {
		case "a":
			if len(p) != 2 {
				fmt.Println("Accumulator reset syntax: r a.unit")
			} else {
				unit, _ := strconv.Atoi(p[1])
				accreset(unit)
			}
		case "c":
			consreset()
		case "d":
			divreset()
		case "f":
			if len(p) != 2 {
				fmt.Println("Function table reset syntax: r f.unit")
			} else {
				unit, _ := strconv.Atoi(p[1])
				ftreset(unit)
			}
		case "i":
			initreset()
		case "m":
			multreset()
		case "p":
			mpreset()
		}
	case "R":
		initreset()
		cycreset()
		mpreset()
		ftreset(0)
		ftreset(1)
		ftreset(2)
		for i := 0; i < 20; i++ {
			accreset(i)
		}
		divreset()
		multreset()
		consreset()
		prreset()
		adreset()
		trayreset()
	case "s":
		if len(f) < 3 {
			fmt.Println("No switch setting")
			break
		}
		p := strings.Split(f[1], ".")
		switch {
		case p[0][0] == 'a':
			if len(p) != 2 {
				fmt.Println("Invalid accumulator switch:", cmd)
			} else {
				unit, _ := strconv.Atoi(p[0][1:])
				accsw[unit-1] <- [2]string{p[1], f[2]}
			}
		case p[0] == "c":
			if len(p) != 2 {
				fmt.Println("Constant switch syntax: s c.switch value")
			} else {
				conssw <- [2]string{p[1], f[2]}
			}
		case p[0] == "cy":
			if len(p) != 2 {
				fmt.Println("Cycling switch syntax: s cy.switch value")
			} else {
				cycsw <- [2]string{p[1], f[2]}
			}
		case p[0] == "d" || p[0] == "ds":
			if len(p) != 2 {
				fmt.Println("Divider switch syntax: s d.switch value")
			} else {
				divsw <- [2]string{p[1], f[2]}
			}
		case p[0][0] == 'f':
			if len(p) != 2 {
				fmt.Println("Function table switch syntax: s funit.switch value", cmd)
			} else {
				unit, _ := strconv.Atoi(p[0][1:])
				ftsw[unit-1] <- [2]string{p[1], f[2]}
			}
		case p[0] == "m":
			if len(p) != 2 {
				fmt.Println("Multiplier switch syntax: s m.switch value")
			} else {
				multsw <- [2]string{p[1], f[2]}
			}
		case p[0] == "p":
			if len(p) != 2 {
				fmt.Println("Programmer switch syntax: s p.switch value")
			} else {
				mpsw <- [2]string{p[1], f[2]}
			}
		case p[0] == "pr":
			if len(p) != 2 {
				fmt.Println("Printer switch syntax: s pr.switch value")
			} else {
				prsw <- [2]string{p[1], f[2]}
			}
		default:
			fmt.Printf("unknown unit for switch: %s\n", p[0])
		}
	case "u":
	case "dt":
	case "pt":
	default:
		if cmd[0] != '#' {
			fmt.Printf("Unknown command: %s\n", cmd)
		}
	}
	return 0
}

func main() {
	var acccyc [20]chan pulse
	var ftcyc [3]chan pulse

	runtime.GOMAXPROCS(1)
	flag.Usage = func () {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [configuration file]\n", os.Args[0])
		flag.PrintDefaults()
	}
	wp := flag.Int("w", 0, "`width` of the simulation window in pixels")
	nogui := flag.Bool("g", false, "run without GUI")
	flag.Parse()
	width = *wp
	if !*nogui {
		go gui()
		ppunch = make(chan string)
	}

	initbut = make(chan int)
	cycsw = make(chan [2]string)
	cycbut = make(chan int)
	mpsw = make(chan [2]string)
	divsw = make(chan [2]string)
	multsw = make(chan [2]string)
	conssw = make(chan [2]string)
	cycout := make(chan pulse)
	cyctrunk := make([]chan pulse, 0, 40)
	initcyc := make(chan pulse)
	mpcyc := make(chan pulse)
	divcyc := make(chan pulse)
	multcyc := make(chan pulse)
	conscyc := make(chan pulse)
	prsw = make(chan [2]string)
	p := append(cyctrunk, initcyc, mpcyc, divcyc, multcyc, conscyc)
	for i := 0; i < 20; i++ {
		accsw[i] = make(chan [2]string)
		acccyc[i] = make(chan pulse)
		p = append(p, acccyc[i])
	}
	for i := 0; i < 3; i++ {
		ftsw[i] = make(chan [2]string)
		ftcyc[i] = make(chan pulse)
		p = append(p, ftcyc[i])
	}
	go fanout(cycout, p)

	go consctl(conssw)
	go mpctl(mpsw)
	go cyclectl(cycsw)
	go divsrctl(divsw)
	go multctl(multsw)
	go prctl(prsw)

	go initiateunit(initcyc, initbut)
	go mpunit(mpcyc)
	go cycleunit(cycout, cycbut)
	go divunit(divcyc)
	go multunit(multcyc)
	go consunit(conscyc)
	for i := 0; i < 20; i++ {
		go accctl(i, accsw[i])
		go accunit(i, acccyc[i])
	}
	for i := 0; i < 3; i++ {
		go ftctl(i, ftsw[i])
		go ftunit(i, ftcyc[i])
	}

	if flag.NArg() >= 1 {
		// Seriously ugly hack to give other goprocs time to get initialized
		time.Sleep(100*time.Millisecond)
		proccmd("l " + flag.Arg(0))
	}

	sc := bufio.NewScanner(os.Stdin)
	fmt.Print("> ")
	for sc.Scan() {
		if proccmd(sc.Text()) < 0 {
			break
		}
		fmt.Print("> ")
	}
}
