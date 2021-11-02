package main

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/google/go-github/v39/github"
	"golang.org/x/oauth2"
)

type tagResponse struct {
	Release  string `json:"release"`
	HasError bool   `json:"has_error"`
	Error    string `json:"erorr"`
}

var ghClient *github.Client

func getIndex(c *fiber.Ctx) error {
	return c.SendString("Try accessing /:owner/:repo")
}

func getLive(c *fiber.Ctx) error {
	return c.SendString("OK")
}

func getLatestReleaseTag(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	ctx := context.Background()
	release, res, err := ghClient.Repositories.GetLatestRelease(ctx, owner, repo)
	if err != nil {
		if bytes.Contains(c.Context().URI().QueryString(), []byte("plain")) {
			return c.Status(res.StatusCode).SendString("")
		}
		return c.Status(res.StatusCode).JSON(tagResponse{
			HasError: true,
			Error:    err.Error(),
		})
	}
	if bytes.Contains(c.Context().URI().QueryString(), []byte("plain")) {
		return c.SendString(release.GetTagName())
	}
	return c.JSON(tagResponse{
		HasError: false,
		Release:  release.GetTagName(),
	})
}

func githubClient(ctx context.Context) *github.Client {
	token := os.Getenv("GITHUB_TOKEN")
	var oauthClient *http.Client
	if token != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		oauthClient = oauth2.NewClient(ctx, ts)
	}
	return github.NewClient(oauthClient)
}

func main() {
	ctx := context.Background()
	ghClient = githubClient(ctx)

	app := fiber.New()
	app.Get("/", getIndex)
	app.Get("/live", getLive)
	app.Get("/:owner/:repo", getLatestReleaseTag)
	log.Fatal(app.Listen("0.0.0.0:8080"))
}
