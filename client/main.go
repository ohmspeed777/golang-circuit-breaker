package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/afex/hystrix-go/hystrix"
	"github.com/gofiber/fiber/v2"
)

func init() {
	hystrix.ConfigureCommand("api", hystrix.CommandConfig{
		Timeout:                500,
		RequestVolumeThreshold: 1,
		ErrorPercentThreshold:  100,
		SleepWindow:            15000,
	})

	hystrixStream := hystrix.NewStreamHandler()
	hystrixStream.Start()
	go http.ListenAndServe(":8002", hystrixStream)
}

func main() {
	app := fiber.New()

	app.Get("/api", api)

	app.Listen(":8081")
}

func api(c *fiber.Ctx) error {
	resp := make(chan string, 1)

	hystrix.Go("api", func() error {
		res, err := http.Get("http://localhost:8000/api")
		if err != nil {
			return err
		}
		defer res.Body.Close()

		data, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}

		msg := string(data)
		fmt.Println(msg)

		resp <- msg
		return errors.New("text")
	}, func(err error) error {
		fmt.Println(err)

		if err == hystrix.ErrCircuitOpen {
			resp <- "error"
		}

		return err
	})

	result := <-resp
	return c.SendString(result)
}
