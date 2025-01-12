// Copyright 2012 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build wasm

package main

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"html"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"strings"
	"syscall/js"

	"rsc.io/qr" // QR code library for generating codes
)

//go:embed pjw.png
var pjwPNG []byte

var (
	doc js.Value // JS document

	// checkboxes
	checkRand    js.Value
	checkData    js.Value
	checkDither  js.Value
	checkControl js.Value

	inputURL js.Value // URL input box
	inputECL js.Value // ECL dropdown
)

var pic = &Image{
	File:    pjwPNG,
	Dx:      4,
	Dy:      4,
	URL:     "https://research.swtch.com/qart",
	Version: 6,
	Mask:    2,
}

// Directional movement functions
func up()       { pic.Dy++ }
func down()     { pic.Dy-- }
func left()     { pic.Dx++ }
func right()    { pic.Dx-- }
func ibigger()  { pic.Size++ }
func ismaller() { pic.Size-- }
func rotate()   { pic.Rotation = (pic.Rotation + 1) & 3 }
func bigger() {
	if pic.Version < 8 {
		pic.Version++
	}
}
func smaller() {
	if pic.Version > 1 {
		pic.Version--
	}
}

func setImage(id string, img []byte) {
	doc.Call("getElementById", id).Set("src", "data:image/png;base64,"+base64.StdEncoding.EncodeToString(img))
}

func setErr(err error) {
	doc.Call("getElementById", "err-output").Set("innerHTML", html.EscapeString(err.Error()))
}

// Update QR code and UI
func update() {
    // 1. Read text and ECL from DOM
    text := inputURL.Get("value").String()
    eclStr := inputECL.Get("value").String()

    var level qr.Level
    switch eclStr {
    case "M":
        level = qr.M
    case "Q":
        level = qr.Q
    case "H":
        level = qr.H
    default:
        // If none match, default to L
        level = qr.L
    }

    // 2. Store these in pic
    pic.ECL = level
    pic.URL = text
    pic.Rand = checkRand.Get("checked").Bool()
    pic.OnlyDataBits = checkData.Get("checked").Bool()
    pic.Dither = checkDither.Get("checked").Bool()
    pic.SaveControl = checkControl.Get("checked").Bool()

    // 3. Actually generate the QArt code
    pngBytes, err := pic.Encode()
    if err != nil {
        setErr(err)
        return
    }

    // 4. Display the result in the browser
    setImage("img-output", pngBytes)
    doc.Call("getElementById", "img-download").Set("href",
        "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes))
}

func funcOf(f func()) js.Func {
	return js.FuncOf(func(_ js.Value, _ []js.Value) any {
		f()
		return nil
	})
}

func main() {
    doc = js.Global().Get("document")

    checkRand = doc.Call("getElementById", "rand")
    checkData = doc.Call("getElementById", "data")
    checkDither = doc.Call("getElementById", "dither")
    checkControl = doc.Call("getElementById", "control")

    inputURL = doc.Call("getElementById", "url")
    inputECL = doc.Call("getElementById", "ecl")

    setImage("arrow-right", Arrow(48, 0))
    setImage("arrow-up", Arrow(48, 1))
    setImage("arrow-left", Arrow(48, 2))
    setImage("arrow-down", Arrow(48, 3))

    setImage("arrow-smaller", Arrow(20, 2))
    setImage("arrow-bigger", Arrow(20, 0))

    setImage("arrow-ismaller", Arrow(20, 2))
    setImage("arrow-ibigger", Arrow(20, 0))

    doc.Call("getElementById", "loading").Get("style").Set("display", "none")
    doc.Call("getElementById", "wasm1").Get("style").Set("display", "block")
    doc.Call("getElementById", "wasm2").Get("style").Set("display", "block")

    if img, err := pic.Src(); err == nil {
        setImage("img-src", img)
    } else {
        setErr(err)
    }

    doc.Call("getElementById", "left").Set("onclick", js.FuncOf(func(this js.Value, args []js.Value) any {
        left()
        update()
        return nil
    }))
    doc.Call("getElementById", "right").Set("onclick", js.FuncOf(func(this js.Value, args []js.Value) any {
        right()
        update()
        return nil
    }))
    doc.Call("getElementById", "up").Set("onclick", js.FuncOf(func(this js.Value, args []js.Value) any {
        up()
        update()
        return nil
    }))
    doc.Call("getElementById", "down").Set("onclick", js.FuncOf(func(this js.Value, args []js.Value) any {
        down()
        update()
        return nil
    }))
    doc.Call("getElementById", "smaller").Set("onclick", js.FuncOf(func(this js.Value, args []js.Value) any {
        smaller()
        update()
        return nil
    }))
    doc.Call("getElementById", "bigger").Set("onclick", js.FuncOf(func(this js.Value, args []js.Value) any {
        bigger()
        update()
        return nil
    }))
    doc.Call("getElementById", "ibigger").Set("onclick", js.FuncOf(func(this js.Value, args []js.Value) any {
        ibigger()
        update()
        return nil
    }))
    doc.Call("getElementById", "ismaller").Set("onclick", js.FuncOf(func(this js.Value, args []js.Value) any {
        ismaller()
        update()
        return nil
    }))
    doc.Call("getElementById", "rotate").Set("onclick", js.FuncOf(func(this js.Value, args []js.Value) any {
        rotate()
        update()
        return nil
    }))

    doc.Call("getElementById", "redraw").Set("onclick", js.FuncOf(func(this js.Value, args []js.Value) any {
        update()
        return nil
    }))
    inputECL.Call("addEventListener", "change", js.FuncOf(func(this js.Value, args []js.Value) any {
        update()
        return nil
    }))

    // File upload callback (unchanged)
    doc.Call("getElementById", "upload-input").Call("addEventListener", "change",
        js.FuncOf(func(this js.Value, args []js.Value) any {
            files := this.Get("files")
            if files.Get("length").Int() != 1 {
                return nil
            }
            r := js.Global().Get("FileReader").New()
            var cb js.Func
            cb = js.FuncOf(func(this js.Value, args []js.Value) any {
                _, enc, _ := strings.Cut(r.Get("result").String(), ";base64,")
                data, err := base64.StdEncoding.DecodeString(enc)
                defer cb.Release()
                if err != nil {
                    setErr(err)
                    return nil
                }
                _, _, err = image.Decode(bytes.NewReader(data))
                if err != nil {
                    setErr(err)
                    return nil
                }
                pic.SetFile(data)
                img, err := pic.Src()
                if err != nil {
                    setErr(err)
                    return nil
                }
                setImage("img-src", img)
                update()
                return nil
            })
            r.Call("addEventListener", "load", cb)
            r.Call("readAsDataURL", files.Index(0))
            return nil
        }))

    // Initial update to render the default state
    update()
    select {}
}

