package dxweb

import (
	"sync"
	"time"

	"github.com/gopherjs/gopherjs/js"
	"github.com/mrmiguu/jsutil"
)

var game *js.Object

func init() {
	<-jsutil.Load("assets/js/phaser.min.js")
	game = js.Global.Get("Phaser").Get("Game").New(800, 600, nil, nil, js.M{"preload": preload, "create": create})
}

func preload() {
}

func create() {
	// game.Get("stage").Set("backgroundColor", "#182d3b")
	game.Get("load").Get("onFileComplete").Call("add", fileComplete)
}

var x = 32
var y = 80

var testl sync.Mutex

func fileComplete(_, cacheKey *js.Object) {
	go func() {
		testl.Lock()
		var newImage = game.Get("add").Call("image", x, y, cacheKey)
		newImage.Set("alpha", 0)
		fade := game.Get("add").Call("tween", newImage).Call("to", js.M{"alpha": 1}, 2500)
		fade.Set("frameBased", true)
		fade.Call("start")
		newImage.Get("scale").Call("set", 0.3)
		x += newImage.Get("width").Int() + 20
		if x > 700 {
			x = 32
			y += 332
		}
		time.Sleep(1 * time.Second)
		testl.Unlock()
	}()
}

func Image(url string) {
	game.Get("load").Call("image", url, url)
	game.Get("load").Call("start")
}
