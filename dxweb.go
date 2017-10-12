package dxweb

import (
	"sync"

	"github.com/gopherjs/gopherjs/js"
	"github.com/mrmiguu/jsutil"
)

var (
	Width  = 800
	Height = 600

	start sync.Once

	phaser *js.Object
	game   *js.Object
	load   *js.Object

	imagel sync.Mutex
	images = map[string]imageLoader{}
)

func init() {
	style := js.Global.Get("document").Get("body").Get("style")
	style.Set("background", "#000000")
	style.Set("margin", 0)
	<-jsutil.Load("assets/js/phaser.min.js")
}

func run() {
	f, c := jsutil.C()
	phaser = js.Global.Get("Phaser")
	game = phaser.Get("Game").New(Width, Height, nil, nil, js.M{"create": f})
	<-c

	game.Get("canvas").Set("oncontextmenu", func(e *js.Object) { e.Call("preventDefault") })
	mode := phaser.Get("ScaleManager").Get("SHOW_ALL")
	scale := game.Get("scale")
	scale.Set("scaleMode", mode)
	scale.Set("fullScreenScaleMode", mode)
	scale.Set("pageAlignHorizontally", true)
	scale.Set("pageAlignVertically", true)

	load = game.Get("load")
	load.Get("onFileComplete").Call("add", func(_, key *js.Object) {
		obj := game.Get("add").Call("image", game.Get("world").Get("centerX"), game.Get("world").Get("centerY"), key)
		obj.Set("alpha", 0)
		obj.Get("anchor").Call("setTo", 0.5, 0.5)

		imagel.Lock()
		img := images[key.String()]
		imagel.Unlock()

		wh := <-img.wh
		obj.Set("width", wh[0])
		obj.Set("height", wh[1])
		img.js <- obj
	})
}

func tween(obj *js.Object, to js.M, ms ...int) {
	move := game.Get("add").Call("tween", obj)
	move.Call("to", to, getMS(ms...))
	move.Set("frameBased", true)
	f, c := jsutil.C()
	move.Get("onComplete").Call("add", f)
	move.Call("start")
	<-c
}

func getMS(ms ...int) int {
	if len(ms) > 1 {
		jsutil.Alert("too many arguments")
	}
	if len(ms) > 0 {
		if ms[0] < 1 {
			jsutil.Alert("negative or zero ms")
		}
		return ms[0]
	}
	return 1
}

type imageLoader struct {
	wh <-chan [2]int
	js chan<- *js.Object
}

type Image struct {
	key           string
	width, height int
	js            *js.Object
}

func LoadImage(url string, width, height int) <-chan Image {
	start.Do(run)

	load.Call("image", url, url)
	load.Call("start")

	wh := make(chan [2]int, 1)
	wh <- [2]int{width, height}
	obj := make(chan *js.Object, 1)
	imagel.Lock()
	images[url] = imageLoader{wh, obj}
	imagel.Unlock()

	c := make(chan Image)
	go func() {
		c <- Image{
			key:    url,
			width:  width,
			height: height,
			js:     <-obj,
		}
	}()
	return c
}

func (i *Image) Pos() (int, int) {
	return i.js.Get("x").Int(), i.js.Get("y").Int()
}

func (i *Image) Size() (int, int) {
	return i.js.Get("width").Int(), i.js.Get("height").Int()
}

func (i *Image) Move(x, y int, ms ...int) {
	tween(i.js, js.M{"x": x, "y": y}, ms...)
}

func (i *Image) Resize(width, height int, ms ...int) {
	tween(i.js, js.M{"width": width, "height": height}, ms...)
}

func (i *Image) Show(b bool, ms ...int) {
	a := 0
	if b {
		a = 1
	}
	tween(i.js, js.M{"alpha": a}, ms...)
}
