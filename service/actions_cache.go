package service

import (
	"errors"
	"fmt"
	"log"
	"math"
	"net/url"
	"strconv"

	"github.com/actions/gh-actions-cache/types"
	gh "github.com/cli/go-gh"
	"github.com/cli/go-gh/pkg/api"
	ghRepo "github.com/cli/go-gh/pkg/repository"
)

type ArtifactCacheService interface {
	GetCacheUsage() float64
	ListCaches(queryParams url.Values) types.ListApiResponse
	DeleteCaches(queryParams url.Values) int
	ListAllCaches(queryParams url.Values, key string) []types.ActionsCache
}

type ArtifactCache struct {
	HttpClient api.RESTClient
	repo       ghRepo.Repository
}

func NewArtifactCache(repo ghRepo.Repository, command string, version string) ArtifactCacheService {
	opts := api.ClientOptions{
		Host:    repo.Host(),
		Headers: map[string]string{"User-Agent": fmt.Sprintf("gh-actions-cache/%s/%s", version, command)},
	}
	restClient, err := gh.RESTClient(&opts)
	if err != nil {
		log.Fatal(err)
	}
	return &ArtifactCache{HttpClient: restClient, repo: repo}
}

func (a *ArtifactCache) GetCacheUsage() float64 {
	pathComponent := fmt.Sprintf("repos/%s/%s/actions/cache/usage", a.repo.Owner(), a.repo.Name())
	var apiResults types.RepoLevelUsageApiResponse
	err := a.HttpClient.Get(pathComponent, &apiResults)
	if err != nil {
		log.Fatal(err)
	}

	return apiResults.ActiveCacheSizeInBytes
}

func (a *ArtifactCache) ListCaches(queryParams url.Values) types.ListApiResponse {
	pathComponent := fmt.Sprintf("repos/%s/%s/actions/caches", a.repo.Owner(), a.repo.Name())
	var apiResults types.ListApiResponse
	err := a.HttpClient.Get(pathComponent+"?"+queryParams.Encode(), &apiResults)
	if err != nil {
		log.Fatal(err)
	}

	return apiResults
}

func (a *ArtifactCache) DeleteCaches(queryParams url.Values) int {
	pathComponent := fmt.Sprintf("repos/%s/%s/actions/caches", a.repo.Owner(), a.repo.Name())
	var apiResults types.DeleteApiResponse
	err := a.HttpClient.Delete(pathComponent+"?"+queryParams.Encode(), &apiResults)
	if err != nil {
		var httpError api.HTTPError
		if errors.As(err, &httpError) && httpError.StatusCode == 404 {
			return 0
		} else {
			log.Fatal(err)
		}
	}
	return apiResults.TotalCount
}

func (a *ArtifactCache) ListAllCaches(queryParams url.Values, key string) []types.ActionsCache {
	var listApiResponse types.ListApiResponse
	listApiResponse = a.ListCaches(queryParams)
	caches := listApiResponse.ActionsCaches
	totalCaches := listApiResponse.TotalCount
	if totalCaches > 100 {
		for page := 2; page <= int(math.Ceil(float64(listApiResponse.TotalCount)/100)); page++ {
			queryParams.Set("page", strconv.Itoa(page))
			listApiResponse = a.ListCaches(queryParams)
			caches = append(caches, listApiResponse.ActionsCaches...)
		}
	}
	return caches
}
