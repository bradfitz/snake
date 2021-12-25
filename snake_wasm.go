package main

import (
	"bytes"
	"container/list"
	"fmt"
	"log"
	"math/rand"
	"syscall/js"
	"time"
)

type gridState byte

const (
	Empty gridState = 0
	Black gridState = 'B'
	Apple gridState = 'A'
)

type coord struct {
	x, y uint8
}

type gameState struct {
	board  [16][16]gridState
	trail  *list.List // of coord
	dx, dy int8
	maxLen int
	speed  time.Duration
}

func newGame() *gameState {
	st := &gameState{
		maxLen: 4,
		trail:  list.New(),
		speed:  time.Second / 2,
	}
	st.board[0][0] = Black
	st.board[15][15] = Black
	st.ensnake(8, 8)
	st.dx = 1
	st.setApple()
	return st
}

func (st *gameState) handleClick(click coord) {
	cur := st.trail.Front().Value.(coord)
	switch {
	case st.dx == 0: // moving up or down, so turn left or right
		if click.x < cur.x { // go left
			st.dx, st.dy = -1, 0
			return
		}
		if click.x > cur.x { // go right
			st.dx, st.dy = 1, 0
			return
		}
	case st.dy == 0: // moving left or right, so turn up or down
		if click.y < cur.y { // go up
			st.dx, st.dy = 0, -1
			return
		}
		if click.y > cur.y { // go down
			st.dx, st.dy = 0, 1
			return
		}
	}
}

func (st *gameState) tick() (stillAlive bool) {
	cur := st.trail.Front().Value.(coord)
	nx, ny := int8(cur.x)+st.dx, int8(cur.y)+st.dy
	if nx < 0 || nx > 15 || ny < 0 || ny > 15 {
		log.Printf("oob; dead")
		return false
	}
	at := st.board[nx][ny]
	if at == Black {
		log.Printf("new (%v, %v) is black; dead", nx, ny)
		return false
	}
	// log.Printf("moved to (%v, %v)", nx, ny)
	if at == Apple {
		st.maxLen += 2
		st.speed -= 20 * time.Millisecond
		const min = 50 * time.Millisecond
		if st.speed < min {
			st.speed = min
		}
		log.Printf("nom; now speed %v, maxLen %v", st.speed, st.maxLen)
	}
	st.ensnake(uint8(nx), uint8(ny))
	for st.trail.Len() > st.maxLen {
		back := st.trail.Remove(st.trail.Back()).(coord)
		x, y := back.x, back.y
		st.board[x][y] = Empty
		if e := doc.Call("getElementById", fmt.Sprintf("p%d", y*16+x)); !e.IsNull() {
			e.Set("src", fmt.Sprintf("http://198.49.126.%d/white.png", y*16+x))
		}
	}
	if at == Apple {
		st.setApple()
	}
	return true
}

func (st *gameState) setApple() {
	for {
		x, y := uint8(rand.Intn(16)), uint8(rand.Intn(16))
		if st.board[x][y] != Empty {
			continue
		}
		st.board[x][y] = Apple
		if e := doc.Call("getElementById", fmt.Sprintf("p%d", y*16+x)); !e.IsNull() {
			e.Set("src", fmt.Sprintf("http://198.49.126.%d/apple.png", y*16+x))
		}
		return
	}
}

func (st *gameState) ensnake(x, y uint8) {
	st.trail.PushFront(coord{x, y})
	st.board[x][y] = Black
	if e := doc.Call("getElementById", fmt.Sprintf("p%d", y*16+x)); !e.IsNull() {
		e.Set("src", fmt.Sprintf("http://198.49.126.%d/black.png", y*16+x))
	}
}

func (st *gameState) initDOM() {
	board := doc.Call("getElementById", "board")
	var buf bytes.Buffer
	buf.WriteString("<body><h1>snake/24</h1><p>IP addresses are precious. Make the most of them. Each grid pixel below is served from a different IP.</p><p>(This might seem like a waste, but this port wasn't being used anyway?)</p><p><b>Instructions:</b> use arrows (or WASD, or clicking/touching direction) to move the snake and eat them apples.</p>")
	oct := -1
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			oct++
			blocked := oct == 0 || oct == 255
			ip := oct
			if blocked {
				ip = 1
			}
			color := "white"
			if blocked {
				color = "black"
			} else {
				switch st.board[x][y] {
				case Black:
					color = "black"
				case Apple:
					color = "apple"
				}
			}
			fmt.Fprintf(&buf, "<img id=p%v src='http://198.49.126.%v/%s.png' width=5%% height=5%%>", oct, ip, color)
		}
		buf.WriteString("<br>\n")
	}
	buf.WriteString("<hr><i>Brad Fitzpatrick, &lt;<a href='https://twitter.com/bradfitz/'>@bradfitz</a>&gt; [<a href='https://twitter.com/bradfitz/status/1474273311316013060'>discuss</a>]")
	board.Set("innerHTML", buf.String())

	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			oct := y*16 + x
			x, y := x, y
			doc.Call("getElementById", fmt.Sprintf("p%d", oct)).Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				select {
				case clickc <- coord{uint8(x), uint8(y)}:
					//log.Printf("clicked (%v, %v)", x, y)
				default:
					log.Printf("clicked (%v, %v); ignored", x, y)
				}
				return true
			}))
		}
	}
}

var doc = js.Global().Get("document")

var clickc = make(chan coord, 1)

func main() {
	log.SetFlags(0)
	log.Printf("snake/24")
	rand.Seed(time.Now().UnixNano())
	doc.Set("bgColor", "#cccccc")
	doc.Get("body").Set("innerHTML", `<div id=board></div>`)

	st := newGame()
	st.initDOM()

	keyCode := make(chan string, 10)
	doc.Call("addEventListener", "keydown", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		e := args[0]
		keyCode <- e.Get("code").String()
		return true
	}))

	timer := time.NewTimer(st.speed)
	for {
		select {
		case key := <-keyCode:
			switch key {
			case "KeyW", "ArrowUp":
				st.dx, st.dy = 0, -1
			case "KeyS", "ArrowDown":
				st.dx, st.dy = 0, 1
			case "KeyA", "ArrowLeft":
				st.dx, st.dy = -1, 0
			case "KeyD", "ArrowRight":
				st.dx, st.dy = 1, 0
			default:
				log.Printf("unknown key %q", key)
				continue
			}
			log.Printf("key %q, dx=%v, dy=%v", key, st.dx, st.dy)
		case click := <-clickc:
			st.handleClick(click)
		case <-timer.C:
			if !st.tick() {
				js.Global().Call("alert", "you lose")
				st = newGame()
				st.initDOM()
			}
			timer = time.NewTimer(st.speed)
		}
	}
}
