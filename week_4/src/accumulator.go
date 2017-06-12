package main

import (
	"fmt"
	"strconv"
)

var units [20]struct {
	α, β, γ, δ, ε, A, S chan pulse
	ctlterm [20]chan pulse
	inff1, inff2 [12]bool
	opsw [12]byte
	clrsw [12]bool
	rptsw [8]byte
	sigfig int
	sc byte
	val [10]byte
	decff [10]bool
	sign bool
	h50	bool
	rep int
	whichrp bool
	change chan int
}

func accstat(unit int) string {
	var s string

	if units[unit].sign {
		s = "M "
	} else {
		s = "P "
	}
	for i := 9; i >= 0; i-- {
		s += fmt.Sprintf("%d", units[unit].val[i])
	}
	s += " "
	for i := 9; i >= 0; i-- {
		s += b2is(units[unit].decff[i])
	}
	s += fmt.Sprintf(" %d ", units[unit].rep)
	for _, f := range units[unit].inff2 {
		s += b2is(f)
	}
	return s
}

func accreset(unit int) {
	u := &units[unit]
	u.α = nil
	u.β = nil
	u.γ = nil
	u.δ = nil
	u.ε = nil
	for i := 0; i < 12; i++ {
		u.ctlterm[i] = nil
		u.inff1[i] = false
		u.inff2[i] = false
		u.opsw[i] = 0
		u.clrsw[i] = false
	}
	for i := 0; i < 8; i++ {
		u.rptsw[i] = 0
	}
	u.sigfig = 10
	u.sc = 0
	u.h50 = false
	u.rep = 0
	u.whichrp = false
	accclear(unit)
	u.change <- 1
}

func accclear(acc int) {
	for i := 0; i < 10; i++ {
		units[acc].val[i] = 0
		units[acc].decff[i] = false
	}
	if units[acc].sigfig < 10 {
		units[acc].val[9-units[acc].sigfig] = 5
	}
	units[acc].sign = false
}

func accplug(unit int, jack string, ch chan pulse) {
	jacks := [20]string{"1i", "2i", "3i", "4i", "5i", "5o", "6i", "6o", "7i", "7o",
		"8i", "8o", "9i", "9o", "10i", "10o", "11i", "11o", "12i", "12o"}

	switch {
	case jack == "α", jack == "a", jack == "alpha":
		units[unit].α = ch
	case jack == "β", jack == "b", jack == "beta":
		units[unit].β = ch
	case jack == "γ", jack == "g", jack == "gamma":
		units[unit].γ = ch
	case jack == "δ", jack == "d", jack == "delta":
		units[unit].δ = ch
	case jack == "ε", jack == "e", jack == "epsilon":
		units[unit].ε = ch
	case jack == "A":
		units[unit].A = ch
	case jack == "S":
		units[unit].S = ch
	case jack[0] == 'I':
	default:
		foundjack := false
		for i, j := range jacks {
			if j == jack {
				units[unit].ctlterm[i] = ch
				foundjack = true
				break
			}
		}
		if !foundjack {
			fmt.Println("Invalid jack:", jack, "on accumulator", unit+1)
		}
	}
	units[unit].change <- 1
}

func accctl(unit int, ch chan [2]string) {
	for {
		ctl :=<- ch
		prog, _ := strconv.Atoi(ctl[0][2:])
		prog--
		switch ctl[0][:2] {
		case "op":
			switch ctl[1] {
			case "α", "a", "alpha":
				units[unit].opsw[prog] = 0
			case "β", "b", "beta":
				units[unit].opsw[prog] = 1
			case "γ", "g", "gamma":
				units[unit].opsw[prog] = 2
			case "δ", "d", "delta":
				units[unit].opsw[prog] = 3
			case "ε", "e", "epsilon":
				units[unit].opsw[prog] = 4
			case "0":
				units[unit].opsw[prog] = 5
			case "A":
				units[unit].opsw[prog] = 6
			case "AS":
				units[unit].opsw[prog] = 7
			case "S":
				units[unit].opsw[prog] = 8
			default:
				fmt.Println("Invalid operation code:", ctl[1], "on unit",
					unit+1, "program", prog+1)
			}
		case "cc":
			switch ctl[1] {
			case "0":
				units[unit].clrsw[prog] = false
			case "C", "c":
				units[unit].clrsw[prog] = true
			default:
				fmt.Println("Invalid clear/correct setting:", ctl[1], "on unit",
					unit+1, "program", prog+1)
			}
		case "rp":
			rpt, err := strconv.Atoi(ctl[1])
			if err == nil {
				units[unit].rptsw[prog-4] = byte(rpt-1)
			} else {
				fmt.Println("Invalid repeat count:", ctl[1], "on unit",
					unit+1, "program", prog+1)
			}
		case "sf":
			n, _ := strconv.Atoi(ctl[1])
			units[unit].sigfig = 10 - n
		case "sc":
			switch ctl[1] {
			case "0":
				units[unit].sc = 0
			case "SC", "sc":
				units[unit].sc = 1
			default:
				fmt.Println("Invalid selective clear setting:", ctl[1],
					"on unit", unit+1)
			}
		}
	}
}

func accrecv(unit, dat int) {
	u := &units[unit]
	for i := 0; i < 10; i++ {
		if dat & 1 == 1 {
			u.val[i]++
			if u.val[i] >= 10 {
				u.decff[i] = true
				u.val[i] -= 10
			}
		}
		dat >>= 1
	}
	if dat & 1 == 1 {
		u.sign = !u.sign
	}
}

func accunit(unit int, cyctrunk chan pulse) {
	u := &units[unit]
	u.change = make(chan int)
	u.sigfig = 10
	go accunit2(unit)

	resp := make(chan int)
	for {
		p :=<- cyctrunk
		cyc := p.val
		nprog := 0
		curprog := 0
		for i := 0; i < 12; i++ {
			if u.inff2[i] {
				curprog |= 1 << uint(u.opsw[i])
				nprog++
			}
		}
		if nprog > 1 && cyc & Ccg != 0{
			fmt.Printf("Multiple programs active on accumulator %d: %v\n", unit, u.inff2)
		}
		if cyc & Cpp != 0 {
			for i := 0; i < 4; i++ {
				u.inff2[i] = false
			}
			if u.h50 {
				u.rep++
				rstrep := false
				for i := 4; i < 12; i++ {
					if u.inff2[i] && u.rep == int(u.rptsw[i-4]) + 1 {
						u.inff2[i] = false
						rstrep = true
						t := (i - 4) * 2 + 5
						if u.ctlterm[t] != nil {
							u.ctlterm[t] <- pulse{1, resp}
							<- resp
						}
					}
				}
				if rstrep {
					u.rep = 0
					u.h50 = false
				}
			}
		} else if cyc & Ccg != 0 {
			u.whichrp = false
			if curprog & 0x1f != 0 || unit == 10 && Multl || unit == 12 && Multr ||
					unit == 1 && divsr.qα ||
					unit == 2 && (divsr.nα || divsr.nβ || divsr.nγ || divsr.npγ) ||
					unit == 4 && (divsr.dα || divsr.dβ || divsr.dγ || divsr.dpγ) ||
					unit == 6 && divsr.sα {
				for i := 0; i < 9; i++ {
					if u.decff[i] {
						u.val[i+1]++
						if u.val[i+1] == 10 {
							u.val[i+1] = 0
							u.decff[i+1] = true
						}
					}
				}
				if u.decff[9] {
					u.sign = !u.sign
				}
			} else {
				for j := 0; j < 12; j++ {
					if u.inff2[j] && u.clrsw[j] && u.opsw[j] >= byte(6) &&
								(j < 4 || u.rep == int(u.rptsw[j-4])) ||
							unit == 2 && divsr.nac ||
							unit == 6 && divsr.sac ||
							unit == 1 && (divsr.divadap == 0 && divsr.ans2 ||
								divsr.divadap == 2 && (divsr.ans1 || divsr.ans2)) ||
							unit == 4 && (divsr.sradap == 0 && divsr.ans4 ||
								divsr.sradap == 2 && (divsr.ans3 || divsr.ans4)) {
						for i := 0; i < 10; i++ {
							u.val[i] = byte(0)
						}
						u.sign = false
						break
					}
				}
			}
		} else if cyc & Scg != 0 {
			if u.sc == 1 {
				accclear(unit)
			}
		} else if cyc & Rp != 0 {
			if u.whichrp {
				/*
				 * Ugly hack to avoid races
				 */
				for i := 0; i < 12; i++ {
					if u.inff1[i] {
						u.inff1[i] = false
						u.inff2[i] = true
						if i >= 4 {
							u.h50 = true
						}
					}
				}
			} else {
				for i := 0; i < 10; i++ {
					u.decff[i] = false
				}
				u.whichrp = true
			}
		} else if cyc & Tenp != 0 && (curprog & 0x1c0 != 0 ||
				unit == 2 && divsr.nac ||
				unit == 4 && (divsr.da || divsr.ds) ||
				unit == 6 && divsr.sac ||
				unit == 1 && (divsr.ans1 || divsr.ans2) ||
				unit == 4 && (divsr.ans3 || divsr.ans4)) {
			for i := 0; i < 10; i++ {
				u.val[i]++
				if u.val[i] == 10 {
					u.val[i] = 0
					u.decff[i] = true
				}
			}
		} else if cyc & Ninep != 0 && (curprog & 0x1c0 != 0 ||
				unit == 2 && divsr.nac ||
				unit == 4 && (divsr.da || divsr.ds) ||
				unit == 6 && divsr.sac ||
				unit == 1 && (divsr.ans1 || divsr.ans2) ||
				unit == 4 && (divsr.ans3 || divsr.ans4)) {
			if curprog & 0xc0 != 0 || unit == 2 && divsr.nac ||
					unit == 4 && divsr.da || unit == 6 && divsr.sac ||
					unit == 1 && (divsr.ans1 || divsr.divadap == 0 && divsr.ans2) ||
					unit == 4 && (divsr.ans3 || divsr.sradap == 0 && divsr.ans4) {
				if u.A != nil {
					n := 0
					for i := 0; i < 10; i++ {
						if u.decff[i] {
							n |= 1 << uint(i)
						}
					}
					if u.sign {
						n |= 1 << 10
					}
					if n != 0 {
						u.A <- pulse{n, resp}
						<- resp
					}
				}
			}
			if curprog & 0x180 != 0 || unit == 4 && divsr.ds ||
					unit == 1 && divsr.ans2 && divsr.divadap != 0 ||
					unit == 4 && divsr.ans4 && divsr.sradap != 0 {
				if u.S != nil {
					n := 0
					for i := 0; i < 10; i++ {
						if !u.decff[i] {
							n |= 1 << uint(i)
						}
					}
					if !u.sign {
						n |= 1 << 10
					}
					if n != 0 {
						u.S <- pulse{n, resp}
						<- resp
					}
				}
			}
		} else if cyc & Onepp != 0 {
			if unit == 2 && (divsr.nγ || divsr.npγ) && divsr.ds {
				u.val[0]++
				if u.val[0] > 9 {
					u.val[0] = 0
					u.decff[0] = true
				}
			}
			for i := 0; i < 12; i++ {
				if u.inff2[i] && u.opsw[i] < 5 && u.clrsw[i] {
					u.val[0]++
					if u.val[0] > 9 {
						u.val[0] = 0
						u.decff[0] = true
					}
					break
				}
			}
			if curprog & 0x180 != 0 && u.sigfig > 0 && u.S != nil {
				u.S <- pulse{1 << uint(10 - u.sigfig), resp}
				<- resp
			}
		}
		if p.resp != nil {
			p.resp <- 1
		}
	}
}


func accunit2(unit int) {
	var dat, prog pulse

	u := &units[unit]
	for {
		select {
		case <- u.change:
		case dat =<- u.α:
			if unit == 10 && Multl || unit == 12 && Multr ||
					unit == 2 && divsr.nα || unit == 4 && divsr.dα ||
					unit == 1 && divsr.qα || unit == 6 && divsr.sα {
				accrecv(unit, dat.val)
			} else {
				for i := 0; i < 12; i++ {
					if u.inff2[i] && u.opsw[i] == 0 {
						accrecv(unit, dat.val)
						break
					}
				}
			}
			if dat.resp != nil {
				dat.resp <- 1
			}
		case dat =<- u.β:
			if unit == 2 && divsr.nβ || unit == 4 && divsr.dβ {
				accrecv(unit, dat.val)
			} else {
				for i := 0; i < 12; i++ {
					if u.inff2[i] && u.opsw[i] == 1 {
						accrecv(unit, dat.val)
						break
					}
				}
			}
			if dat.resp != nil {
				dat.resp <- 1
			}
		case dat =<- u.γ:
			if unit == 2 && (divsr.nγ || divsr.npγ) ||
					unit == 4 && (divsr.dγ || divsr.dpγ){
				accrecv(unit, dat.val)
			} else {
				for i := 0; i < 12; i++ {
					if u.inff2[i] && u.opsw[i] == 2 {
						accrecv(unit, dat.val)
						break
					}
				}
			}
			if dat.resp != nil {
				dat.resp <- 1
			}
		case dat =<- u.δ:
			for i := 0; i < 12; i++ {
				if u.inff2[i] && u.opsw[i] == 3 {
					accrecv(unit, dat.val)
					break
				}
			}
			if dat.resp != nil {
				dat.resp <- 1
			}
		case dat =<- u.ε:
			for i := 0; i < 12; i++ {
				if u.inff2[i] && u.opsw[i] == 4 {
					accrecv(unit, dat.val)
					break
				}
			}
			if dat.resp != nil {
				dat.resp <- 1
			}
		case prog =<- u.ctlterm[0]:
			if prog.val == 1 {
				u.inff1[0] = true
			}
			if prog.resp != nil {
				prog.resp <- 1
			}
		case prog =<- u.ctlterm[1]:
			if prog.val == 1 {
				u.inff1[1] = true
			}
			if prog.resp != nil {
				prog.resp <- 1
			}
		case prog =<- u.ctlterm[2]:
			if prog.val == 1 {
				u.inff1[2] = true
			}
			if prog.resp != nil {
				prog.resp <- 1
			}
		case prog =<- u.ctlterm[3]:
			if prog.val == 1 {
				u.inff1[3] = true
			}
			if prog.resp != nil {
				prog.resp <- 1
			}
		case prog =<- u.ctlterm[4]:
			if prog.val == 1 {
				u.inff1[4] = true
			}
			if prog.resp != nil {
				prog.resp <- 1
			}
		case prog =<- u.ctlterm[6]:
			if prog.val == 1 {
				u.inff1[5] = true
			}
			if prog.resp != nil {
				prog.resp <- 1
			}
		case prog =<- u.ctlterm[8]:
			if prog.val == 1 {
				u.inff1[6] = true
			}
			if prog.resp != nil {
				prog.resp <- 1
			}
		case prog =<- u.ctlterm[10]:
			if prog.val == 1 {
				u.inff1[7] = true
			}
			if prog.resp != nil {
				prog.resp <- 1
			}
		case prog =<- u.ctlterm[12]:
			if prog.val == 1 {
				u.inff1[8] = true
			}
			if prog.resp != nil {
				prog.resp <- 1
			}
		case prog =<- u.ctlterm[14]:
			if prog.val == 1 {
				u.inff1[9] = true
			}
			if prog.resp != nil {
				prog.resp <- 1
			}
		case prog =<- u.ctlterm[16]:
			if prog.val == 1 {
				u.inff1[10] = true
			}
			if prog.resp != nil {
				prog.resp <- 1
			}
		case prog =<- u.ctlterm[18]:
			if prog.val == 1 {
				u.inff1[11] = true
			}
			if prog.resp != nil {
				prog.resp <- 1
			}
		}
	}
}
