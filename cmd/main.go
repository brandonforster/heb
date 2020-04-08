/*******************************************************************************
 * Copyright 2020 Dell Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under the License
 * is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express
 * or implied. See the License for the specific language governing permissions and limitations under
 * the License.
 *******************************************************************************/

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/wav"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func translateInterruptToCancel(ctx context.Context, wg *sync.WaitGroup, cancel context.CancelFunc) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		signalStream := make(chan os.Signal)
		defer func() {
			signal.Stop(signalStream)
			close(signalStream)
		}()
		signal.Notify(signalStream, os.Interrupt, syscall.SIGTERM)
		select {
		case <-signalStream:
			cancel()
			return
		case <-ctx.Done():
			return
		}
	}()
}

func playSound() {
	f, err := os.Open("167.wav")
	if err != nil {
		return
	}
	streamer, format, err := wav.Decode(f)
	if err != nil {
		fmt.Println("wav.Decode err: ", err.Error())
		return
	}
	_ = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
	speaker.Play(streamer)
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	translateInterruptToCancel(ctx, &wg, cancel)

	wg.Add(1)
	go func() {
		defer wg.Done()

		timeout := 0
		for {
			for i := 0; i < timeout; i++ {
				select {
				case <-ctx.Done():
					return
				default:
					time.Sleep(time.Second)
				}
			}
			timeout = 25 + rand.Intn(10)

			fmt.Print(time.Now(), " -- ")

			resp, err := http.Get("https://www.heb.com/commerce-api/v1/timeslot/timeslots?store_id=659&days=20&fulfillment_type=pickup")
			if err != nil {
				fmt.Println("http.Get err: ", err.Error())
				continue
			}

			defer func() {
				_ = resp.Body.Close()
			}()

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Println("ioutil.ReadAll err: ", err.Error())
				continue
			}

			var result map[string]interface{}
			err = json.Unmarshal(body, &result)
			if err != nil {
				fmt.Println("ioutil.ReadAll err: ", err.Error())
				continue
			}

			value, ok := result["items"]
			if !ok {
				fmt.Println("items missing from response")
				continue
			}

			items, ok := value.([]interface{})
			if !ok {
				fmt.Println("type assertion failed")
				continue
			}

			if len(items) == 0 {
				fmt.Println("no slots (", string(body), ")")
				continue
			}

			playSound()
			fmt.Println("********************** ", items)
		}
	}()

	wg.Wait()
}
