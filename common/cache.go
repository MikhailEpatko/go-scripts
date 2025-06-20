package common

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

// setCacheUpdateEnabled - отключение/включение обновления кешей плюсометра
func SetCacheUpdateEnabled(enable bool) error {
	for _, cache := range []string{"table1", "table2"} {
		var err = func() error {
			var url = fmt.Sprintf("%s/enable-cache-updating?cache=%s&enable=%v", UrlPrefix, cache, enable)
			var request, err = http.NewRequest(http.MethodPut, url, nil)
			if err != nil {
				return fmt.Errorf("setCacheUpdateEnabled: request creating error: %w", err)
			}
			request.Header.Add("Content-Type", "application/json")
			request.Header.Add("accept", "application/json")
			resp, err := HttpClient.Do(request)
			if err != nil {
				return fmt.Errorf("setCacheUpdateEnabled: request sending error: %w", err)
			}
			defer func() {
				var err = resp.Body.Close()
				if err != nil {
					log.Printf("%s: setCacheUpdateEnabled: closing response body error: %v", cache, err)
				}
			}()
			if resp.StatusCode != 200 {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					log.Printf("%s: setCacheUpdateEnabled: response body: %s\n", cache, string(body))
					return fmt.Errorf("setCacheUpdateEnabled: reading response body error: %w", err)
				}
			}
			return nil
		}()
		if err != nil {
			return err
		}
	}
	return nil
}
