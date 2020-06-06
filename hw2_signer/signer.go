package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

func ExecutePipeline(jobs ...job) {
	var in, out chan interface{}
	wg := new(sync.WaitGroup)
	wg.Add(len(jobs))

	for _, j := range jobs {
		in = out
		out = make(chan interface{}, MaxInputDataLen)

		go func(j job, in, out chan interface{}) {
			defer wg.Done()
			defer close(out)

			j(in, out)
		}(j, in, out)
	}

	wg.Wait()
}

func sendHashedStr(in, out chan interface{}, hashFn func(string) string) {
	wg := new(sync.WaitGroup)

	for data := range in {
		wg.Add(1)

		go func(str string) {
			defer wg.Done()

			out <- hashFn(str)
		}(fmt.Sprintf("%v", data))
	}

	wg.Wait()
}

func SingleHash(in, out chan interface{}) {
	mux := new(sync.Mutex)

	sendHashedStr(in, out, func(data string) string {
		h1 := make(chan string)
		go func() {
			h1 <- DataSignerCrc32(data)
		}()

		h2 := make(chan string)
		go func() {
			mux.Lock()
			md5Hash := DataSignerMd5(data)
			mux.Unlock()

			h2 <- DataSignerCrc32(md5Hash)
		}()

		return fmt.Sprintf("%s~%s", <-h1, <-h2)
	})
}

func MultiHash(in, out chan interface{}) {
	sendHashedStr(in, out, func(data string) string {
		multiHash := make([]string, 6)
		mux := new(sync.Mutex)
		wg := new(sync.WaitGroup)

		for i := 0; i < len(multiHash); i++ {
			wg.Add(1)

			go func(n int) {
				defer wg.Done()

				hash := DataSignerCrc32(fmt.Sprintf("%d%s", n, data))
				mux.Lock()
				multiHash[n] = hash
				mux.Unlock()
			}(i)
		}

		wg.Wait()
		return strings.Join(multiHash, "")
	})
}

func CombineResults(in, out chan interface{}) {
	var results []string

	for data := range in {
		results = append(results, fmt.Sprintf("%v", data))
	}

	sort.Strings(results)
	out <- strings.Join(results, "_")
}