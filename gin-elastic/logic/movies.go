package logic

import (
	"bufio"
	"context"
	"encoding/json"
	"es4gophers/domain"
	"fmt"
	"os"
	"sync"
)

func LoadMoviesFromFile(ctx context.Context) context.Context {

	const (
		concurrency = 5
		moviesFile  = "movies.json"
	)

	var (
		movies    []domain.Movie
		waitGroup = new(sync.WaitGroup)
		workQueue = make(chan string)
		mutex     = &sync.Mutex{}
	)

	go func() {
		moviesFile, err := os.Open(moviesFile)
		if err != nil {
			panic(err)
		}
		defer moviesFile.Close()
		scanner := bufio.NewScanner(moviesFile)
		for scanner.Scan() {
			workQueue <- scanner.Text()
		}
		close(workQueue)
	}()

	for i := 0; i < concurrency; i++ {
		waitGroup.Add(1)
		go func(workQueue chan string, waitGroup *sync.WaitGroup) {
			for entry := range workQueue {
				movieRaw := domain.MovieRaw{}
				err := json.Unmarshal([]byte(entry), &movieRaw)
				if err == nil {
					movie := func(movieRaw domain.MovieRaw) domain.Movie {
						return domain.Movie{
							Title:       movieRaw.Title,
							Year:        movieRaw.Year,
							RunningTime: movieRaw.Info.RunningTime,
							ReleaseDate: movieRaw.Info.ReleaseDate,
							Rating:      movieRaw.Info.Rating,
							Genres:      movieRaw.Info.Genres,
							Actors:      movieRaw.Info.Actors,
							Directors:   movieRaw.Info.Directors,
						}
					}(movieRaw)
					mutex.Lock()
					movies = append(movies, movie)
					mutex.Unlock()
				}
			}
			waitGroup.Done()
		}(workQueue, waitGroup)
	}

	waitGroup.Wait()

	fmt.Printf("✅ Movies loaded from the file: %d \n", len(movies))
	return context.WithValue(ctx, domain.MoviesKey, movies)

}
