//go:build tcell || windows

package tui

import (
	"testing"

	"github.com/gdamore/tcell"
	"github.com/junegunn/fzf/src/util"
)

func assert(t *testing.T, context string, got interface{}, want interface{}) bool {
	if got == want {
		return true
	} else {
		t.Errorf("%s = (%T)%v, want (%T)%v", context, got, got, want, want)
		return false
	}
}

// Test the handling of the tcell keyboard events.
func TestGetCharEventKey(t *testing.T) {
	if util.ToTty() {
		// This test is skipped when output goes to terminal, because it causes
		// some glitches:
		// - output lines may not start at the beginning of a row which makes
		//   the output unreadable
		// - terminal may get cleared which prevents you from seeing results of
		//   previous tests
		// Good ways to prevent the glitches are piping the output to a pager
		// or redirecting to a file. I've found `less +G` to be trouble-free.
		t.Skip("Skipped because this test misbehaves in terminal, pipe to a pager or redirect output to a file to run it safely.")
	} else if testing.Verbose() {
		// I have observed a behaviour when this test outputted more than 8192
		// bytes (32*256) into the 'less' pager, both the go's test executable
		// and the pager hanged. The go's executable was blocking on printing.
		// I was able to create minimal working example of that behaviour, but
		// that example hanged after 12256 bytes (32*(256+127)).
		t.Log("If you are piping this test to a pager and it hangs, make the pager greedy for input, e.g. 'less +G'.")
	}

	if !HasFullscreenRenderer() {
		t.Skip("Can't test FullscreenRenderer.")
	}

	// construct test cases
	type giveKey struct {
		Type tcell.Key
		Char rune
		Mods tcell.ModMask
	}
	type wantKey = Event
	type testCase struct {
		giveKey
		wantKey
	}
	/*
		Some test cases are marked "fabricated". It means that giveKey value
		is valid, but it is not what you get when you press the keys. For
		example Ctrl+C will NOT give you tcell.KeyCtrlC, but tcell.KeyETX
		(End-Of-Text character, causing SIGINT).
		I was trying to accompany the fabricated test cases with real ones.

		Some test cases are marked "unhandled". It means that giveKey.Type
		is not present in tcell.go source code. It can still be handled via
		implicit or explicit alias.

		If not said otherwise, test cases are for US keyboard.

		(tabstop=44)
	*/
	tests := []testCase{

		// section 1: Ctrl+(Alt)+[a-z]
		{giveKey{tcell.KeyCtrlA, rune(tcell.KeyCtrlA), tcell.ModCtrl}, wantKey{CtrlA, 0, nil}},
		{giveKey{tcell.KeyCtrlC, rune(tcell.KeyCtrlC), tcell.ModCtrl}, wantKey{CtrlC, 0, nil}}, // fabricated
		{giveKey{tcell.KeyETX, rune(tcell.KeyETX), tcell.ModCtrl}, wantKey{CtrlC, 0, nil}},     // this is SIGINT (Ctrl+C)
		{giveKey{tcell.KeyCtrlZ, rune(tcell.KeyCtrlZ), tcell.ModCtrl}, wantKey{CtrlZ, 0, nil}}, // fabricated
		// KeyTab is alias for KeyTAB
		{giveKey{tcell.KeyCtrlI, rune(tcell.KeyCtrlI), tcell.ModCtrl}, wantKey{Tab, 0, nil}}, // fabricated
		{giveKey{tcell.KeyTab, rune(tcell.KeyTab), tcell.ModNone}, wantKey{Tab, 0, nil}},     // unhandled, actual "Tab" keystroke
		{giveKey{tcell.KeyTAB, rune(tcell.KeyTAB), tcell.ModNone}, wantKey{Tab, 0, nil}},     // fabricated, unhandled
		// KeyEnter is alias for KeyCR
		{giveKey{tcell.KeyCtrlM, rune(tcell.KeyCtrlM), tcell.ModNone}, wantKey{CtrlM, 0, nil}}, // actual "Enter" keystroke
		{giveKey{tcell.KeyCR, rune(tcell.KeyCR), tcell.ModNone}, wantKey{CtrlM, 0, nil}},       // fabricated, unhandled
		{giveKey{tcell.KeyEnter, rune(tcell.KeyEnter), tcell.ModNone}, wantKey{CtrlM, 0, nil}}, // fabricated, unhandled
		// Ctrl+Alt keys
		{giveKey{tcell.KeyCtrlA, rune(tcell.KeyCtrlA), tcell.ModCtrl | tcell.ModAlt}, wantKey{CtrlAlt, 'a', nil}},                  // fabricated
		{giveKey{tcell.KeyCtrlA, rune(tcell.KeyCtrlA), tcell.ModCtrl | tcell.ModAlt | tcell.ModShift}, wantKey{CtrlAlt, 'a', nil}}, // fabricated

		// section 2: Ctrl+[ \]_]
		{giveKey{tcell.KeyCtrlSpace, rune(tcell.KeyCtrlSpace), tcell.ModCtrl}, wantKey{CtrlSpace, 0, nil}}, // fabricated
		{giveKey{tcell.KeyNUL, rune(tcell.KeyNUL), tcell.ModNone}, wantKey{CtrlSpace, 0, nil}},             // fabricated, unhandled
		{giveKey{tcell.KeyRune, ' ', tcell.ModCtrl}, wantKey{CtrlSpace, 0, nil}},                           // actual Ctrl+' '
		{giveKey{tcell.KeyCtrlBackslash, rune(tcell.KeyCtrlBackslash), tcell.ModCtrl}, wantKey{CtrlBackSlash, 0, nil}},
		{giveKey{tcell.KeyCtrlRightSq, rune(tcell.KeyCtrlRightSq), tcell.ModCtrl}, wantKey{CtrlRightBracket, 0, nil}},
		{giveKey{tcell.KeyCtrlCarat, rune(tcell.KeyCtrlCarat), tcell.ModShift | tcell.ModCtrl}, wantKey{CtrlCaret, 0, nil}}, // fabricated
		{giveKey{tcell.KeyRS, rune(tcell.KeyRS), tcell.ModShift | tcell.ModCtrl}, wantKey{CtrlCaret, 0, nil}},               // actual Ctrl+Shift+6 (i.e. Ctrl+^) keystroke
		{giveKey{tcell.KeyCtrlUnderscore, rune(tcell.KeyCtrlUnderscore), tcell.ModShift | tcell.ModCtrl}, wantKey{CtrlSlash, 0, nil}},

		// section 3: (Alt)+Backspace2
		// KeyBackspace2 is alias for KeyDEL = 0x7F (ASCII) (allegedly unused by Windows)
		// KeyDelete = 0x2E (VK_DELETE constant in Windows)
		// KeyBackspace is alias for KeyBS = 0x08 (ASCII) (implicit alias with KeyCtrlH)
		{giveKey{tcell.KeyBackspace2, 0, tcell.ModNone}, wantKey{BSpace, 0, nil}}, // fabricated
		{giveKey{tcell.KeyBackspace2, 0, tcell.ModAlt}, wantKey{AltBS, 0, nil}},   // fabricated
		{giveKey{tcell.KeyDEL, 0, tcell.ModNone}, wantKey{BSpace, 0, nil}},        // fabricated, unhandled
		{giveKey{tcell.KeyDelete, 0, tcell.ModNone}, wantKey{Del, 0, nil}},
		{giveKey{tcell.KeyDelete, 0, tcell.ModAlt}, wantKey{Del, 0, nil}},
		{giveKey{tcell.KeyBackspace, 0, tcell.ModNone}, wantKey{Invalid, 0, nil}},                                                  // fabricated, unhandled
		{giveKey{tcell.KeyBS, 0, tcell.ModNone}, wantKey{Invalid, 0, nil}},                                                         // fabricated, unhandled
		{giveKey{tcell.KeyCtrlH, 0, tcell.ModNone}, wantKey{Invalid, 0, nil}},                                                      // fabricated, unhandled
		{giveKey{tcell.KeyCtrlH, rune(tcell.KeyCtrlH), tcell.ModNone}, wantKey{BSpace, 0, nil}},                                    // actual "Backspace" keystroke
		{giveKey{tcell.KeyCtrlH, rune(tcell.KeyCtrlH), tcell.ModAlt}, wantKey{AltBS, 0, nil}},                                      // actual "Alt+Backspace" keystroke
		{giveKey{tcell.KeyDEL, rune(tcell.KeyDEL), tcell.ModCtrl}, wantKey{BSpace, 0, nil}},                                        // actual "Ctrl+Backspace" keystroke
		{giveKey{tcell.KeyCtrlH, rune(tcell.KeyCtrlH), tcell.ModShift}, wantKey{BSpace, 0, nil}},                                   // actual "Shift+Backspace" keystroke
		{giveKey{tcell.KeyCtrlH, 0, tcell.ModCtrl | tcell.ModAlt}, wantKey{BSpace, 0, nil}},                                        // actual "Ctrl+Alt+Backspace" keystroke
		{giveKey{tcell.KeyCtrlH, 0, tcell.ModCtrl | tcell.ModShift}, wantKey{BSpace, 0, nil}},                                      // actual "Ctrl+Shift+Backspace" keystroke
		{giveKey{tcell.KeyCtrlH, rune(tcell.KeyCtrlH), tcell.ModShift | tcell.ModAlt}, wantKey{AltBS, 0, nil}},                     // actual "Shift+Alt+Backspace" keystroke
		{giveKey{tcell.KeyCtrlH, 0, tcell.ModCtrl | tcell.ModAlt | tcell.ModShift}, wantKey{BSpace, 0, nil}},                       // actual "Ctrl+Shift+Alt+Backspace" keystroke
		{giveKey{tcell.KeyCtrlH, rune(tcell.KeyCtrlH), tcell.ModCtrl}, wantKey{CtrlH, 0, nil}},                                     // actual "Ctrl+H" keystroke
		{giveKey{tcell.KeyCtrlH, rune(tcell.KeyCtrlH), tcell.ModCtrl | tcell.ModAlt}, wantKey{CtrlAlt, 'h', nil}},                  // fabricated "Ctrl+Alt+H" keystroke
		{giveKey{tcell.KeyCtrlH, rune(tcell.KeyCtrlH), tcell.ModCtrl | tcell.ModShift}, wantKey{CtrlH, 0, nil}},                    // actual "Ctrl+Shift+H" keystroke
		{giveKey{tcell.KeyCtrlH, rune(tcell.KeyCtrlH), tcell.ModCtrl | tcell.ModAlt | tcell.ModShift}, wantKey{CtrlAlt, 'h', nil}}, // fabricated "Ctrl+Shift+Alt+H" keystroke

		// section 4: (Alt+Shift)+Key(Up|Down|Left|Right)
		{giveKey{tcell.KeyUp, 0, tcell.ModNone}, wantKey{Up, 0, nil}},
		{giveKey{tcell.KeyDown, 0, tcell.ModAlt}, wantKey{AltDown, 0, nil}},
		{giveKey{tcell.KeyLeft, 0, tcell.ModShift}, wantKey{SLeft, 0, nil}},
		{giveKey{tcell.KeyRight, 0, tcell.ModShift | tcell.ModAlt}, wantKey{AltSRight, 0, nil}},
		{giveKey{tcell.KeyUpLeft, 0, tcell.ModNone}, wantKey{Invalid, 0, nil}},    // fabricated, unhandled
		{giveKey{tcell.KeyUpRight, 0, tcell.ModNone}, wantKey{Invalid, 0, nil}},   // fabricated, unhandled
		{giveKey{tcell.KeyDownLeft, 0, tcell.ModNone}, wantKey{Invalid, 0, nil}},  // fabricated, unhandled
		{giveKey{tcell.KeyDownRight, 0, tcell.ModNone}, wantKey{Invalid, 0, nil}}, // fabricated, unhandled
		{giveKey{tcell.KeyCenter, 0, tcell.ModNone}, wantKey{Invalid, 0, nil}},    // fabricated, unhandled
		// section 5: (Insert|Home|Delete|End|PgUp|PgDn|BackTab|F1-F12)
		{giveKey{tcell.KeyInsert, 0, tcell.ModNone}, wantKey{Insert, 0, nil}},
		{giveKey{tcell.KeyF1, 0, tcell.ModNone}, wantKey{F1, 0, nil}},
		// section 6: (Ctrl+Alt)+'rune'
		{giveKey{tcell.KeyRune, 'a', tcell.ModNone}, wantKey{Rune, 'a', nil}},
		{giveKey{tcell.KeyRune, 'a', tcell.ModCtrl}, wantKey{Rune, 'a', nil}}, // fabricated
		{giveKey{tcell.KeyRune, 'a', tcell.ModAlt}, wantKey{Alt, 'a', nil}},
		{giveKey{tcell.KeyRune, 'A', tcell.ModAlt}, wantKey{Alt, 'A', nil}},
		{giveKey{tcell.KeyRune, '`', tcell.ModAlt}, wantKey{Alt, '`', nil}},
		/*
			"Input method" in Windows Language options:
			US: "US Keyboard" does not generate any characters (and thus any events) in Ctrl+Alt+[a-z] range
			CS: "Czech keyboard"
			DE: "German keyboard"

			Note that right Alt is not just `tcell.ModAlt` on foreign language keyboards, but it is the AltGr `tcell.ModCtrl|tcell.ModAlt`.
		*/
		{giveKey{tcell.KeyRune, '{', tcell.ModCtrl | tcell.ModAlt}, wantKey{Rune, '{', nil}}, // CS: Ctrl+Alt+b = "{" // Note that this does not interfere with CtrlB, since the "b" is replaced with "{" on OS level
		{giveKey{tcell.KeyRune, '$', tcell.ModCtrl | tcell.ModAlt}, wantKey{Rune, '$', nil}}, // CS: Ctrl+Alt+ů = "$"
		{giveKey{tcell.KeyRune, '~', tcell.ModCtrl | tcell.ModAlt}, wantKey{Rune, '~', nil}}, // CS: Ctrl+Alt++ = "~"
		{giveKey{tcell.KeyRune, '`', tcell.ModCtrl | tcell.ModAlt}, wantKey{Rune, '`', nil}}, // CS: Ctrl+Alt+ý,Space = "`" // this is dead key, space is required to emit the char

		{giveKey{tcell.KeyRune, '{', tcell.ModCtrl | tcell.ModAlt}, wantKey{Rune, '{', nil}}, // DE: Ctrl+Alt+7 = "{"
		{giveKey{tcell.KeyRune, '@', tcell.ModCtrl | tcell.ModAlt}, wantKey{Rune, '@', nil}}, // DE: Ctrl+Alt+q = "@"
		{giveKey{tcell.KeyRune, 'µ', tcell.ModCtrl | tcell.ModAlt}, wantKey{Rune, 'µ', nil}}, // DE: Ctrl+Alt+m = "µ"

		// section 7: Esc
		// KeyEsc and KeyEscape are aliases for KeyESC
		{giveKey{tcell.KeyEsc, rune(tcell.KeyEsc), tcell.ModNone}, wantKey{ESC, 0, nil}},               // fabricated
		{giveKey{tcell.KeyESC, rune(tcell.KeyESC), tcell.ModNone}, wantKey{ESC, 0, nil}},               // unhandled
		{giveKey{tcell.KeyEscape, rune(tcell.KeyEscape), tcell.ModNone}, wantKey{ESC, 0, nil}},         // fabricated, unhandled
		{giveKey{tcell.KeyESC, rune(tcell.KeyESC), tcell.ModCtrl}, wantKey{ESC, 0, nil}},               // actual Ctrl+[ keystroke
		{giveKey{tcell.KeyCtrlLeftSq, rune(tcell.KeyCtrlLeftSq), tcell.ModCtrl}, wantKey{ESC, 0, nil}}, // fabricated, unhandled

		// section 8: Invalid
		{giveKey{tcell.KeyRune, 'a', tcell.ModMeta}, wantKey{Rune, 'a', nil}}, // fabricated
		{giveKey{tcell.KeyF24, 0, tcell.ModNone}, wantKey{Invalid, 0, nil}},
		{giveKey{tcell.KeyHelp, 0, tcell.ModNone}, wantKey{Invalid, 0, nil}},   // fabricated, unhandled
		{giveKey{tcell.KeyExit, 0, tcell.ModNone}, wantKey{Invalid, 0, nil}},   // fabricated, unhandled
		{giveKey{tcell.KeyClear, 0, tcell.ModNone}, wantKey{Invalid, 0, nil}},  // unhandled, actual keystroke Numpad_5 with Numlock OFF
		{giveKey{tcell.KeyCancel, 0, tcell.ModNone}, wantKey{Invalid, 0, nil}}, // fabricated, unhandled
		{giveKey{tcell.KeyPrint, 0, tcell.ModNone}, wantKey{Invalid, 0, nil}},  // fabricated, unhandled
		{giveKey{tcell.KeyPause, 0, tcell.ModNone}, wantKey{Invalid, 0, nil}},  // unhandled

	}
	r := NewFullscreenRenderer(&ColorTheme{}, false, false)
	r.Init()

	// run and evaluate the tests
	for _, test := range tests {
		// generate key event
		giveEvent := tcell.NewEventKey(test.giveKey.Type, test.giveKey.Char, test.giveKey.Mods)
		_screen.PostEventWait(giveEvent)
		t.Logf("giveEvent = %T{key: %v, ch: %q (%[3]v), mod: %#04b}\n", giveEvent, giveEvent.Key(), giveEvent.Rune(), giveEvent.Modifiers())

		// process the event in fzf and evaluate the test
		gotEvent := r.GetChar()
		// skip Resize events, those are sometimes put in the buffer outside of this test
		for gotEvent.Type == Resize {
			t.Logf("Resize swallowed")
			gotEvent = r.GetChar()
		}
		t.Logf("wantEvent = %T{Type: %v, Char: %q (%[3]v)}\n", test.wantKey, test.wantKey.Type, test.wantKey.Char)
		t.Logf("gotEvent = %T{Type: %v, Char: %q (%[3]v)}\n", gotEvent, gotEvent.Type, gotEvent.Char)

		assert(t, "r.GetChar().Type", gotEvent.Type, test.wantKey.Type)
		assert(t, "r.GetChar().Char", gotEvent.Char, test.wantKey.Char)
	}

	r.Close()
}

/*
Quick reference
---------------

(tabstop=18)
(this is not mapping table, it merely puts multiple constants ranges in one table)

¹) the two columns are each other implicit alias
²) explicit aliases here

%v	section #	tcell ctrl key¹	tcell ctrl char¹	tcell alias²	tui constants	tcell named keys	tcell mods
--	---------	--------------	---------------	-----------	-------------	----------------	----------
0	2	KeyCtrlSpace	KeyNUL = ^@ 		Rune		ModNone
1	1	KeyCtrlA	KeySOH = ^A		CtrlA		ModShift
2	1	KeyCtrlB	KeySTX = ^B		CtrlB		ModCtrl
3	1	KeyCtrlC	KeyETX = ^C		CtrlC
4	1	KeyCtrlD	KeyEOT = ^D		CtrlD		ModAlt
5	1	KeyCtrlE	KeyENQ = ^E		CtrlE
6	1	KeyCtrlF	KeyACK = ^F		CtrlF
7	1	KeyCtrlG	KeyBEL = ^G		CtrlG
8	1	KeyCtrlH	KeyBS = ^H	KeyBackspace	CtrlH		ModMeta
9	1	KeyCtrlI	KeyTAB = ^I	KeyTab	Tab
10	1	KeyCtrlJ	KeyLF = ^J		CtrlJ
11	1	KeyCtrlK	KeyVT = ^K		CtrlK
12	1	KeyCtrlL	KeyFF = ^L		CtrlL
13	1	KeyCtrlM	KeyCR = ^M	KeyEnter	CtrlM
14	1	KeyCtrlN	KeySO = ^N		CtrlN
15	1	KeyCtrlO	KeySI = ^O		CtrlO
16	1	KeyCtrlP	KeyDLE = ^P		CtrlP
17	1	KeyCtrlQ	KeyDC1 = ^Q		CtrlQ
18	1	KeyCtrlR	KeyDC2 = ^R		CtrlR
19	1	KeyCtrlS	KeyDC3 = ^S		CtrlS
20	1	KeyCtrlT	KeyDC4 = ^T		CtrlT
21	1	KeyCtrlU	KeyNAK = ^U		CtrlU
22	1	KeyCtrlV	KeySYN = ^V		CtrlV
23	1	KeyCtrlW	KeyETB = ^W		CtrlW
24	1	KeyCtrlX	KeyCAN = ^X		CtrlX
25	1	KeyCtrlY	KeyEM = ^Y		CtrlY
26	1	KeyCtrlZ	KeySUB = ^Z		CtrlZ
27	7	KeyCtrlLeftSq	KeyESC = ^[	KeyEsc, KeyEscape	ESC
28	2	KeyCtrlBackslash	KeyFS = ^\		CtrlSpace
29	2	KeyCtrlRightSq	KeyGS = ^]		CtrlBackSlash
30	2	KeyCtrlCarat	KeyRS = ^^		CtrlRightBracket
31	2	KeyCtrlUnderscore	KeyUS = ^_		CtrlCaret
32					CtrlSlash
33					Invalid
34					Resize
35					Mouse
36					DoubleClick
37					LeftClick
38					RightClick
39					BTab
40					BSpace
41					Del
42					PgUp
43					PgDn
44					Up
45					Down
46					Left
47					Right
48					Home
49					End
50					Insert
51					SUp
52					SDown
53					SLeft
54					SRight
55					F1
56					F2
57					F3
58					F4
59					F5
60					F6
61					F7
62					F8
63					F9
64					F10
65					F11
66					F12
67					Change
68					BackwardEOF
69					AltBS
70					AltUp
71					AltDown
72					AltLeft
73					AltRight
74					AltSUp
75					AltSDown
76					AltSLeft
77					AltSRight
78					Alt
79					CtrlAlt
..
127	3		  KeyDEL	KeyBackspace2
..
256	6					KeyRune
257	4					KeyUp
258	4					KeyDown
259	4					KeyRight
260	4					KeyLeft
261	8					KeyUpLeft
262	8					KeyUpRight
263	8					KeyDownLeft
264	8					KeyDownRight
265	8					KeyCenter
266	5					KeyPgUp
267	5					KeyPgDn
268	5					KeyHome
269	5					KeyEnd
270	5					KeyInsert
271	5					KeyDelete
272	8					KeyHelp
273	8					KeyExit
274	8					KeyClear
275	8		  			KeyCancel
276	8				  	KeyPrint
277	8					KeyPause
278	5					KeyBacktab
279	5					KeyF1
280	5					KeyF2
281	5					KeyF3
282	5					KeyF4
283	5					KeyF5
284	5					KeyF6
285	5					KeyF7
286	5					KeyF8
287	5					KeyF9
288	5					KeyF10
289	5					KeyF11
290	5					KeyF12
291	8					KeyF13
292	8					KeyF14
293	8					KeyF15
294	8					KeyF16
295	8					KeyF17
296	8					KeyF18
297	8					KeyF19
298	8					KeyF20
299	8					KeyF21
300	8					KeyF22
301	8					KeyF23
302	8					KeyF24
303	8					KeyF25
304	8					KeyF26
305	8					KeyF27
306	8					KeyF28
307	8					KeyF29
308	8					KeyF30
309	8					KeyF31
310	8					KeyF32
311	8					KeyF33
312	8					KeyF34
313	8					KeyF35
314	8					KeyF36
315	8					KeyF37
316	8					KeyF38
317	8					KeyF39
318	8					KeyF40
319	8					KeyF41
320	8					KeyF42
321	8					KeyF43
322	8					KeyF44
323	8					KeyF45
324	8					KeyF46
325	8					KeyF47
326	8					KeyF48
327	8					KeyF49
328	8					KeyF50
329	8					KeyF51
330	8					KeyF52
331	8					KeyF53
332	8					KeyF54
333	8					KeyF55
334	8					KeyF56
335	8					KeyF57
336	8					KeyF58
337	8					KeyF59
338	8					KeyF60
339	8					KeyF61
340	8					KeyF62
341	8					KeyF63
342	8					KeyF64
--	---------	--------------	---------------	-----------	-------------	----------------	----------
%v	section #	tcell ctrl key	tcell ctrl char	tcell alias	tui constants	tcell named keys	tcell mods
*/
