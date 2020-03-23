package main

import (
	"flag"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gdamore/tcell"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

func makebox(s tcell.Screen) {
	w, h := s.Size()

	if w == 0 || h == 0 {
		return
	}

	glyphs := []rune{'@', '#', '&', '*', '=', '%', 'Z', 'A'}

	lx := rand.Int() % w
	ly := rand.Int() % h
	lw := rand.Int() % (w - lx)
	lh := rand.Int() % (h - ly)
	st := tcell.StyleDefault
	gl := ' '
	if s.Colors() > 256 {
		rgb := tcell.NewHexColor(int32(rand.Int() & 0xffffff))
		st = st.Background(rgb)
	} else if s.Colors() > 1 {
		st = st.Background(tcell.Color(rand.Int() % s.Colors()))
	} else {
		st = st.Reverse(rand.Int()%2 == 0)
		gl = glyphs[rand.Int()%len(glyphs)]
	}

	for row := 0; row < lh; row++ {
		for col := 0; col < lw; col++ {
			s.SetCell(lx+col, ly+row, st, gl)
		}
	}
	s.Show()
}

func main() {
	tcell.SetEncodingFallback(tcell.EncodingFallbackASCII)
	s, e := tcell.NewScreen()
	if e != nil {
		fmt.Fprintf(os.Stderr, "%v\n", e)
		os.Exit(1)
	}
	if e = s.Init(); e != nil {
		fmt.Fprintf(os.Stderr, "%v\n", e)
		os.Exit(1)
	}

	s.SetStyle(tcell.StyleDefault.
		Foreground(tcell.ColorBlack).
		Background(tcell.ColorWhite))
	s.Clear()

	quit := make(chan struct{})
	go func() {
		for {
			ev := s.PollEvent()
			switch ev := ev.(type) {
			case *tcell.EventKey:
				switch ev.Key() {
				case tcell.KeyEscape, tcell.KeyEnter:
					close(quit)
					return
				case tcell.KeyCtrlL:
					s.Sync()
				}
			case *tcell.EventResize:
				s.Sync()
			}
		}
	}()

	cnt := 0
	dur := time.Duration(0)

loop:
	for {
		select {
		case <-quit:
			break loop
		case <-time.After(time.Millisecond * 50):
		}
		start := time.Now()
		makebox(s)
		cnt++
		dur += time.Now().Sub(start)
	}

	s.Fini()
	fmt.Printf("Finished %d boxes in %s\n", cnt, dur)
	fmt.Printf("Average is %0.3f ms / box\n", (float64(dur)/float64(cnt))/1000000.0)
}

// func main() {
// 	if err := ui.Init(); err != nil {
// 		log.Fatalf("failed to initialize termui: %v", err)
// 	}
// 	defer ui.Close()
//
// 	width, height := ui.TerminalDimensions()
//
// 	col := false
//
// 	drawFunction := func() {
// 		if col {
// 			colWidth := width / 3
//
// 			p := widgets.NewParagraph()
// 			p.Title = "This is a title that is clearly too long for this window"
// 			p.Text = "This is text that is clearly too long for this window"
// 			p.SetRect(0, 0, colWidth, height)
//
// 			p2 := widgets.NewParagraph()
// 			p2.Title = "This is a title that is clearly too long for this window"
// 			p2.Text = "This is text that is clearly too long for this window"
// 			p2.SetRect(colWidth, 0, colWidth*2, height)
//
// 			p3 := widgets.NewParagraph()
// 			p3.Title = "This is a title that is clearly too long for this window"
// 			p3.Text = "This is text that is clearly too long for this window"
// 			p3.SetRect(colWidth*2, 0, colWidth*3, height)
//
// 			ui.Render(p, p2, p3)
// 		} else {
// 			rowHeight := height / 3
//
// 			p := widgets.NewParagraph()
// 			p.Title = "This is a title that is clearly too long for this window"
// 			p.Text = "This is text that is clearly too long for this window"
// 			p.SetRect(0, 0, width, rowHeight)
//
// 			p2 := widgets.NewParagraph()
// 			p2.Title = "This is a title that is clearly too long for this window"
// 			p2.Text = "This is text that is clearly too long for this window"
// 			p2.SetRect(0, rowHeight, width, rowHeight*2)
//
// 			p3 := widgets.NewParagraph()
// 			p3.Title = "This is a title that is clearly too long for this window"
// 			p3.Text = "This is text that is clearly too long for this window"
// 			p3.SetRect(0, rowHeight*2, width, height)
//
// 			ui.Render(p, p2, p3)
// 		}
//
// 	}
//
// 	uiEvents := ui.PollEvents()
// 	ticker := time.NewTicker(time.Second).C
// 	for {
// 		select {
// 		case e := <-uiEvents:
// 			switch e.ID { // event string/identifier
// 			case "q", "<C-c>": // press 'q' or 'C-c' to quit
// 				return
// 			case "<Space>":
// 				col = !col
// 				// case "<MouseLeft>":
// 				// 	payload := e.Payload.(ui.Mouse)
// 				// 	x, y := payload.X, payload.Y
// 				// case "<Resize>":
// 				// 	payload := e.Payload.(ui.Resize)
// 				// 	width, height = payload.Width, payload.Height
// 				// }
// 				// switch e.Type {
// 				// case ui.KeyboardEvent: // handle all key presses
// 				// 	eventID := e.ID // keypress string
// 				// 	fmt.Printf("<keyboard> eventID: %v", eventID)
// 			}
// 		// use Go's built-in tickers for updating and drawing data
// 		case <-ticker:
// 			drawFunction()
// 		}
// 	}
// }

func realMain() {
	if len(os.Args) < 2 {
		fmt.Println("Please provide the namespace as the first argument")
	}

	namespace := os.Args[1]
	fmt.Printf("Tailing pods in namespace %s\n", namespace)

	var kubeconfig *string
	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	pods, err := clientset.CoreV1().Pods(namespace).List(metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("There are %d pods in the namespace %s\n", len(pods.Items), namespace)

	var wg sync.WaitGroup
	for _, pod := range pods.Items {
		wg.Add(1)
		go echoPodLogs(clientset, namespace, pod.Name, &wg)
	}

	wg.Wait()
}

func echoPodLogs(clientset *kubernetes.Clientset, namespace string, podName string, wg *sync.WaitGroup) {
	defer wg.Done()

	sinceSeconds := int64(30)
	plo := &v1.PodLogOptions{
		Follow:       true,
		SinceSeconds: &sinceSeconds,
	}
	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, plo)
	podLogs, err := req.Stream()
	if err != nil {
		panic("error in opening string")
	}
	defer podLogs.Close()

	for {
		buf := make([]byte, 256)
		n, err := podLogs.Read(buf)
		if err != nil && err == io.EOF {
			fmt.Println("EOF... done")
			break
		}

		if n == 0 {
			continue
		}

		fmt.Printf("[%s] %s\n", podName, string(buf[0:n]))
	}
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
