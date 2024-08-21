package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-queue/queue"
	"github.com/golang-queue/queue/core"
	"github.com/jdxyw/generativeart"
	"github.com/jdxyw/generativeart/arts"
	"github.com/jdxyw/generativeart/common"
	"golang.org/x/exp/maps"
	"image/color"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type jobData struct {
	Id        string
	Generator string
}

var sm sync.Map

func (j *jobData) Bytes() []byte {
	b, _ := json.Marshal(j)
	return b
}

var generatedImages []string

var DRAWINGS = map[string]generativeart.Engine{
	"maze":      arts.NewMaze(10),
	"julia":     arts.NewJulia(func(z complex128) complex128 { return z*z + complex(-0.1, 0.651) }, 40, 1.5, 1.5),
	"randcicle": arts.NewRandCicle(30, 80, 0.2, 2, 10, 30, true),
	"blackhole": arts.NewBlackHole(200, 400, 0.01),
	"janus":     arts.NewJanus(5, 10),
	"random":    arts.NewRandomShape(150),
	"silksky":   arts.NewSilkSky(15, 5),
	"circles":   arts.NewColorCircle2(30),
}

func main() {
	router().Run()
}

func router() *gin.Engine {
	r := gin.Default()
	r.LoadHTMLGlob("templates/*.tmpl")

	q := queue.NewPool(30, queue.WithFn(func(ctx context.Context, m core.QueuedMessage) error {
		j, _ := m.(*jobData)
		json.Unmarshal(m.Bytes(), &j)

		sleepTime := time.Duration(rand.Intn(100)) * time.Second
		time.Sleep(sleepTime)

		path := drawOne(j.Generator)
		sm.Store(j.Id, path)
		fmt.Printf("Stored: %s:%s [%s]\n", j.Id, j.Generator, path)

		return nil
	}))
	imageRoute := r.Group("/image")
	{
		imageRoute.GET("/:generator", func(c *gin.Context) {
			generator := c.Param("generator")
			file := drawOne(generator)
			c.Header("Content-Type", "image/png")
			c.File(file)
		})
	}

	listRoute := r.Group("/list")
	{
		listRoute.GET("/simple", func(c *gin.Context) {
			c.HTML(http.StatusOK, "template.tmpl", gin.H{
				"keys": maps.Keys(DRAWINGS),
			})
		})
	}

	newRoute := r.Group("/new")
	{
		newRoute.GET("/load/:id", func(c *gin.Context) {
			id := c.Param("id")
			path, ok := sm.Load(id)

			if ok {
				fmt.Printf("Found %s for id: %s\n", path, id)
				c.Header("Content-Type", "image/png")
				c.File(fmt.Sprintf("%s", path.(string)))
			} else {
				fmt.Printf("Path not found for id: %s\n", id)
				c.Header("Content-Type", "image/jpg")
				c.Header("Cache-Control", "no-cache")
				c.File("static/loading.jpg")
			}
		})

		newRoute.GET("/:generator", func(c *gin.Context) {
			generator := c.Param("generator")

			newJob := jobData{
				Id:        strconv.Itoa(rand.Int()),
				Generator: generator,
			}

			q.Queue(&newJob)
			generatedImages = append(generatedImages, newJob.Id)
			res := map[string]string{"id": newJob.Id, "url": "http://" + c.Request.Host + "/new/load/" + newJob.Id}
			c.JSON(200, res)
		})
	}

	r.GET("/generated-images-table", func(c *gin.Context) {
		c.HTML(200, "gen-img.tmpl", gin.H{
			"keys": generatedImages,
		})
	})
	return r
}

func drawMany(drawings map[string]generativeart.Engine) {
	for k, _ := range drawings {
		drawOne(k)
	}
}

func drawOne(art string) string {
	//rand.New(rand.NewSource(time.Now().Unix()))
	//rand.Seed(time.Now().Unix())
	c := generativeart.NewCanva(600, 400)
	c.SetColorSchema([]color.RGBA{
		{0xCF, 0x2B, 0x34, 0xFF},
		{0xF0, 0x8F, 0x46, 0xFF},
		{0xF0, 0xC1, 0x29, 0xFF},
		{0x19, 0x6E, 0x94, 0xFF},
		{0x35, 0x3A, 0x57, 0xFF},
	})
	c.SetBackground(common.NavajoWhite)
	c.FillBackground()
	c.SetLineWidth(1.0)
	c.SetLineColor(common.Orange)
	c.Draw(DRAWINGS[art])
	fileName := fmt.Sprintf("%s_%d.png", art, rand.Float64())
	c.ToPNG(fileName)
	return fileName
}
