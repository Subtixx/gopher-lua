package lua

import (
	"os"
	"strings"
	"time"
)

var startedAt time.Time

func init() {
	startedAt = time.Now()
}

func OpenOs(L *LState) int {
	osmod := L.RegisterModule(OsLibName, osFuncs)
	L.Push(osmod)
	return 1
}

func OpenOsBlacklist(L *LState, blacklist ...string) int {
	osmod := L.RegisterModule(OsLibName, blacklistFuncs(osFuncs, blacklist))
	L.Push(osmod)
	return 1
}

var osFuncs = map[string]LGFunction{
	"clock":     osClock,
	"difftime":  osDiffTime,
	"execute":   osExecute,
	"exit":      osExit,
	"date":      osDate,
	"getenv":    osGetEnv,
	"remove":    osRemove,
	"rename":    osRename,
	"setenv":    osSetEnv,
	"setlocale": osSetLocale,
	"time":      osTime,
	"tmpname":   osTmpname,
}

func osClock(L *LState) int {
	L.Push(LNumber(float64(time.Now().Sub(startedAt)) / float64(time.Second)))
	return 1
}

func osDiffTime(L *LState) int {
	L.Push(LNumber(L.CheckInt64(1) - L.CheckInt64(2)))
	return 1
}

func osExecute(L *LState) int {
	var procAttr os.ProcAttr
	procAttr.Files = []*os.File{os.Stdin, os.Stdout, os.Stderr}
	cmd, args := popenArgs(L.CheckString(1))
	args = append([]string{cmd}, args...)
	process, err := os.StartProcess(cmd, args, &procAttr)
	if err != nil {
		L.Push(LNumber(1))
		return 1
	}

	ps, err := process.Wait()
	if err != nil || !ps.Success() {
		L.Push(LNumber(1))
		return 1
	}
	L.Push(LNumber(0))
	return 1
}

func osExit(L *LState) int {
	L.Close()
	os.Exit(L.OptInt(1, 0))
	return 1
}

func osDate(L *LState) int {
	t := time.Now()
	isUTC := false
	cfmt := "%c"
	if L.GetTop() >= 1 {
		cfmt = L.CheckString(1)
		if strings.HasPrefix(cfmt, "!") {
			cfmt = strings.TrimLeft(cfmt, "!")
			isUTC = true
		}
		if L.GetTop() >= 2 {
			t = time.Unix(L.CheckInt64(2), 0)
		}
		if isUTC {
			t = t.UTC()
		}
		if strings.HasPrefix(cfmt, "*t") {
			ret := L.NewTable()
			ret.RawSetString("year", LNumber(t.Year()))
			ret.RawSetString("month", LNumber(t.Month()))
			ret.RawSetString("day", LNumber(t.Day()))
			ret.RawSetString("hour", LNumber(t.Hour()))
			ret.RawSetString("min", LNumber(t.Minute()))
			ret.RawSetString("sec", LNumber(t.Second()))
			ret.RawSetString("wday", LNumber(t.Weekday()+1))
			// TODO yday & dst
			ret.RawSetString("yday", LNumber(0))
			ret.RawSetString("isdst", LFalse)
			L.Push(ret)
			return 1
		}
	}
	L.Push(LString(strftime(t, cfmt)))
	return 1
}

func osGetEnv(L *LState) int {
	v := os.Getenv(L.CheckString(1))
	if len(v) == 0 {
		L.Push(LNil)
	} else {
		L.Push(LString(v))
	}
	return 1
}

func osRemove(L *LState) int {
	err := os.Remove(L.CheckString(1))
	if err != nil {
		L.Push(LNil)
		L.Push(LString(err.Error()))
		return 2
	} else {
		L.Push(LTrue)
		return 1
	}
}

func osRename(L *LState) int {
	err := os.Rename(L.CheckString(1), L.CheckString(2))
	if err != nil {
		L.Push(LNil)
		L.Push(LString(err.Error()))
		return 2
	} else {
		L.Push(LTrue)
		return 1
	}
}

func osSetLocale(L *LState) int {
	// setlocale is not supported
	L.Push(LFalse)
	return 1
}

func osSetEnv(L *LState) int {
	err := os.Setenv(L.CheckString(1), L.CheckString(2))
	if err != nil {
		L.Push(LNil)
		L.Push(LString(err.Error()))
		return 2
	} else {
		L.Push(LTrue)
		return 1
	}
}

func osTime(L *LState) int {
	if L.GetTop() == 0 {
		L.Push(LNumber(time.Now().Unix()))
	} else {
		lv := L.CheckAny(1)
		if lv == LNil {
			L.Push(LNumber(time.Now().Unix()))
		} else {
			tbl, ok := lv.(*LTable)
			if !ok {
				L.TypeError(1, LTTable)
			}
			second := GetIntField(L, tbl, "sec", 0)
			minute := GetIntField(L, tbl, "min", 0)
			hour := GetIntField(L, tbl, "hour", 12)
			day := GetIntField(L, tbl, "day", -1)
			month := GetIntField(L, tbl, "month", -1)
			year := GetIntField(L, tbl, "year", -1)
			isDaylightSaving := GetBoolField(L, tbl, "isdst", false)
			t := time.Date(year, time.Month(month), day, hour, minute, second, 0, time.Local)
			if isDaylightSaving {
				_, offset := t.Zone()
				t = t.Add(time.Duration(offset) * time.Second)
			}
			L.Push(LNumber(t.Unix()))
		}
	}
	return 1
}

func osTmpname(L *LState) int {
	file, err := os.CreateTemp("", "")
	if err != nil {
		L.RaiseError("unable to generate a unique filename")
	}
	//goland:noinspection GoUnhandledErrorResult,GoDfaErrorMayBeNotNil
	file.Close()
	//goland:noinspection GoUnhandledErrorResult,GoDfaErrorMayBeNotNil
	os.Remove(file.Name()) // ignore errors
	//goland:noinspection GoDfaErrorMayBeNotNil
	L.Push(LString(file.Name()))
	return 1
}

//
