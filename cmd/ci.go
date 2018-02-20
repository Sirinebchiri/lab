package cmd

import (
	"fmt"
	"log"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
	"github.com/spf13/cobra"

	"github.com/xanzy/go-gitlab"

	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

// ciCmd represents the ci command
var ciCmd = &cobra.Command{
	Use:   "ci",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		remote, _, err := parseArgsRemote(args)
		if err != nil {
			log.Fatal(err)
		}
		if remote == "" {
			remote = forkedFromRemote
		}

		// See if we're in a git repo or if global is set to determine
		// if this should be a personal snippet
		rn, err := git.PathWithNameSpace(remote)
		if err != nil {
			log.Fatal(err)
		}
		project, err := lab.FindProject(rn)
		if err != nil {
			log.Fatal(err)
		}
		sha, err := git.Sha("HEAD")
		if err != nil {
			log.Fatal(err)
		}
		root = tview.NewPages()
		root.SetBorderPadding(1, 1, 2, 14)

		screen, err = tcell.NewScreen()
		if err != nil {
			log.Fatal(err)
		}
		boxes = make(map[string]*tview.Box)
		jobsCh := make(chan []gitlab.Job)
		go updateJobs(jobsCh, project.ID, sha)
		go func() {
			for {
				jobsLayout(jobsCh, root)
			}
		}()
		a, err := tview.NewApplication()
		if err != nil {
			log.Fatal(err)
		}
		screen = a.Screen()
		if err := a.SetRoot(root, true).Run(); err != nil {
			log.Fatal(err)
		}
	},
}

var (
	screen tcell.Screen
	root   *tview.Pages
	boxes  map[string]*tview.Box
)

func newBox(name string) {
}

func box(key string, x, y, w, h int) *tview.Box {
	fmt.Printf("key: %s, x: %d, y: %d, w: %d, h: %d\n", key, x, y, w, h)
	b, ok := boxes[key]
	if !ok {
		b = tview.NewBox().SetBorder(true)
		boxes[key] = b
	}
	b.SetRect(x, y, w, h)
	root.AddPage(key, b, false, true)
	return b
}

func updateJobs(jobsCh chan []gitlab.Job, pid interface{}, sha string) {
	for {
		jobs, err := lab.CIJobs(pid, sha)
		if err != nil {
			log.Fatal(err)
		}
		// if this deadlocks call g.Update()
		jobsCh <- jobs
		time.Sleep(time.Second * 5)
	}
}

func jobsLayout(jobsCh chan []gitlab.Job, root *tview.Pages) func() error {
	jobs := <-jobsCh
	spew.Dump(jobs)
	px, py, maxX, maxY := root.GetInnerRect()
	fmt.Printf("root x: %d, y: %d, w: %d, h: %d\n", px, py, maxX, maxY)
	var (
		stages    = 0
		lastStage = ""
	)
	// get the number of stages
	for _, j := range jobs {
		if j.Stage != lastStage {
			lastStage = j.Stage
			stages++
		}
	}
	lastStage = ""
	var (
		rowIdx   = 0
		stageIdx = 0
	)
	for _, j := range jobs {
		boxX := px + (maxX / stages * stageIdx)
		if j.Stage != lastStage {
			rowIdx = 0
			stageIdx++
			lastStage = j.Stage
			key := "stage-" + j.Stage

			x, y, w, h := boxX, maxY/2-4, 12, 3
			b := box(key, x, y, w, h)
			b.SetTitle(j.Stage)
		}
	}
	lastStage = jobs[0].Stage
	rowIdx = 0
	stageIdx = 0
	for _, j := range jobs {
		if j.Stage != lastStage {
			rowIdx = 0
			lastStage = j.Stage
			stageIdx++
		}
		fmt.Printf("\nstage: %s, stageIdx: %d, rowIdx: %d\n", j.Stage, stageIdx, rowIdx)
		boxX := px + (maxX / stages * stageIdx)

		key := "jobs-" + j.Name
		x, y, w, h := boxX, maxY/2+(rowIdx*5), 12, 4
		b := box(key, x, y, w, h)
		b.SetTitle(j.Name)
		// The scope of jobs to show, one or array of: created, pending, running,
		// failed, success, canceled, skipped; showing all jobs if none provided
		var statChar rune
		switch j.Status {
		case "success":
			b.SetBorderColor(tcell.ColorGreen)
			statChar = '✔'
		case "failed":
			b.SetBorderColor(tcell.ColorRed)
			statChar = '✘'
		case "running":
			b.SetBorderColor(tcell.ColorBlue)
			statChar = '●'
		case "pending":
			b.SetBorderColor(tcell.ColorYellow)
			statChar = '●'
		}
		retryChar := '⟳'
		_ = retryChar
		b.SetTitle(fmt.Sprintf("%c %s\n", statChar, j.Name))
		rowIdx++

	}
	for i, k := 0, 1; k < len(jobs); i, k = i+1, k+1 {
		v1, ok := boxes["jobs-"+jobs[i].Name]
		if !ok {
			log.Fatal("not okay")
		}
		v2, ok := boxes["jobs-"+jobs[k].Name]
		if !ok {
			log.Fatal("not okay")
		}
		connect(v1, v2)
	}
	return nil
}

func connect(v1 *tview.Box, v2 *tview.Box) {
	x1, y1, w, h := v1.GetRect()
	x2, y2, _, _ := v2.GetRect()

	dx, dy := x2-x1, y2-y1

	// dy != 0 means the last stage had multple jobs
	if dy != 0 && dx != 0 {
		hline(x1+w, y2+h/2, dx-w)
		screen.SetContent(x1+w+2, y2+h/2, '┳', nil, tcell.StyleDefault)
		return
	}
	if dy == 0 {
		hline(x1+w, y1+h/2, dx-w)
		return
	}

	// cells := screen.CellBuffer()
	// tw, _ := screen.Size()

	// '┣' '┫'
	// TODO: fix drawing the last stage (don't draw right side of box)
	// TODO: fix drawing the first stage (don't draw left side of box)

	// Drawing a job in the same stage
	// left of view
	if r, _, _, _ := screen.GetContent(x2-3, y1+h/2); r == '┗' {
		screen.SetContent(x2-3, y1+h/2, '┣', nil, tcell.StyleDefault)
	} else {
		screen.SetContent(x2-3, y1+h/2, '┳', nil, tcell.StyleDefault)
	}

	screen.SetContent(x2-1, y2+h/2, '━', nil, tcell.StyleDefault)
	screen.SetContent(x2-2, y2+h/2, '━', nil, tcell.StyleDefault)
	screen.SetContent(x2-3, y2+h/2, '┗', nil, tcell.StyleDefault)

	// NOTE: unsure what the 2nd arg (y), "-1" is needed for. Maybe due to
	// padding? This showed up after migrating from termbox
	vline(x2-3, y1+h-1, dy-1)
	vline(x2+w+2, y1+h-1, dy-1)

	// right of view
	if r, _, _, _ := screen.GetContent(x2+w+2, y1+h/2); r == '┛' {
		screen.SetContent(x2+w+2, y1+h/2, '┫', nil, tcell.StyleDefault)
	}
	screen.SetContent(x2+w, y2+h/2, '━', nil, tcell.StyleDefault)
	screen.SetContent(x2+w+1, y2+h/2, '━', nil, tcell.StyleDefault)
	screen.SetContent(x2+w+2, y2+h/2, '┛', nil, tcell.StyleDefault)
}

func hline(x, y, l int) {
	for i := 0; i < l; i++ {
		screen.SetContent(x+i, y, '━', nil, tcell.StyleDefault)
		//screen.SetCell(start+i, y1+h1/2, '-', tcell.StyleDefault, tcell.ColorDefault)
	}
}

func vline(x, y, l int) {
	for i := 0; i < l; i++ {
		screen.SetContent(x, y+i, '┃', nil, tcell.StyleDefault)
		//screen.SetCell(x1+w1/2, start+i, '|', tcell.StyleDefault, tcell.ColorDefault)
	}
}

func init() {
	RootCmd.AddCommand(ciCmd)
}
