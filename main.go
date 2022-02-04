package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/go-github/v39/github"
	"github.com/patrickmn/go-cache"
	"golang.org/x/oauth2"
	"golang.org/x/sync/singleflight"
)

type tagAsset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"download_url"`
}

type tagResponse struct {
	Release  string     `json:"release"`
	HasError bool       `json:"has_error"`
	Error    string     `json:"erorr"`
	Assets   []tagAsset `json:"assets"`
}

var ghClient *github.Client
var ghCache *cache.Cache
var sfGroup singleflight.Group

func getIndex(c *fiber.Ctx) error {
	return c.SendString("Try accessing /:owner/:repo")
}

func getLive(c *fiber.Ctx) error {
	return c.SendString("OK")
}

type sfRes struct {
	rel  *github.RepositoryRelease
	code int
}

func fetchLatestReleaseTag(owner, repo string) (*github.RepositoryRelease, int, error) {
	key := fmt.Sprintf("%s/%s", owner, repo)
	if v, found := ghCache.Get(key); found {
		r := v.(*github.RepositoryRelease)
		return r, 200, nil
	}

	r, err, _ := sfGroup.Do(key, func() (interface{}, error) {
		ctx := context.Background()
		release, res, err := ghClient.Repositories.GetLatestRelease(ctx, owner, repo)
		if err != nil {
			return sfRes{release, res.StatusCode}, err
		}
		ghCache.SetDefault(key, release)
		return sfRes{release, res.StatusCode}, err
	})
	if r == nil {
		return nil, 500, fmt.Errorf("something wrong %v", err)
	}
	res := r.(sfRes)
	return res.rel, res.code, err
}

func getLatestReleaseTag(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	release, code, err := fetchLatestReleaseTag(owner, repo)
	if err != nil {
		if code >= 200 && code < 400 {
			code = 500
		}
		if bytes.Contains(c.Context().URI().QueryString(), []byte("plain")) {
			return c.Status(code).SendString("")
		}
		return c.Status(code).JSON(tagResponse{
			HasError: true,
			Error:    err.Error(),
		})
	}
	if bytes.Contains(c.Context().URI().QueryString(), []byte("plain")) {
		return c.SendString(release.GetTagName())
	}
	assets := make([]tagAsset, 0)
	for _, asset := range release.Assets {
		assets = append(assets, tagAsset{
			Name:        asset.GetName(),
			DownloadURL: asset.GetBrowserDownloadURL(),
		})
	}
	return c.JSON(tagResponse{
		HasError: false,
		Release:  release.GetTagName(),
		Assets:   assets,
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

	ghCache = cache.New(5*time.Minute, 10*time.Minute)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	app := fiber.New()
	app.Get("/", getIndex)
	app.Get("/live", getLive)
	app.Get("/:owner/:repo", getLatestReleaseTag)
	log.Fatal(app.Listen("0.0.0.0:" + port))
}
