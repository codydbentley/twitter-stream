package stream

import (
	"bytes"
	"fmt"
	"github.com/codydbentley/twitter-stream/httpclient"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"
)

func TestGetMessages(t *testing.T) {
	client := httpclient.NewHttpClientMock("foobar")
	reader := NewStreamResponseBodyReader()
	instance := NewStream(client, reader)

	messages := instance.GetMessages()

	if messages == nil {
		t.Error("unable to GetMessages. Found nil value!")
	}
}

func TestStopStream(t *testing.T) {
	client := httpclient.NewHttpClientMock("foobar")
	reader := NewStreamResponseBodyReader()
	instance := &Stream{
		unmarshalHook: func(bytes []byte) (interface{}, error) {
			return bytes, nil
		},
		messages:   make(chan StreamMessage),
		done:       make(chan struct{}),
		reader:     reader,
		httpClient: client,
	}

	instance.StopStream()
	result := <-instance.done

	if result != struct{}{} {
		t.Errorf("expected empty struct, got %v", result)
	}
}

func TestStartStream(t *testing.T) {
	var tests = []struct {
		givenMockHttpRequestToStreamReturns func() httpclient.IHttpClient
		givenMockStreamResponseBodyReader   func() IStreamResponseBodyReader
		result                              StreamMessage
	}{
		{
			func() httpclient.IHttpClient {
				mockClient := httpclient.NewHttpClientMock("foobar")
				mockClient.MockGetSearchStream = func(queryParams *url.Values) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       ioutil.NopCloser(bytes.NewReader([]byte("hello"))),
					}, nil
				}
				return mockClient
			},
			func() IStreamResponseBodyReader {
				r := mockStreamResponseBodyReader{}
				r.MockSetStreamResponseBody = func(body io.Reader) {}
				r.MockReadNext = func() ([]byte, error) {
					return []byte("hello"), nil
				}
				return r
			},
			StreamMessage{
				Data: []byte("hello"),
				Err:  nil,
			},
		},
	}

	for i, tt := range tests {
		testName := fmt.Sprintf("TestStartStream (%d)", i)

		t.Run(testName, func(t *testing.T) {
			instance := NewStream(
				tt.givenMockHttpRequestToStreamReturns(),
				tt.givenMockStreamResponseBodyReader(),
			)

			err := instance.StartStream(nil)
			if err != nil {
				t.Errorf("got err when starting stream %v", err)
			}

			result := make(chan StreamMessage)
			go func() {
				for message := range instance.GetMessages() {
					result <- message
				}
			}()
			r := <-result

			expected, _ := tt.result.Data.([]byte)
			res, _ := r.Data.([]byte)

			if string(expected) != string(res) {
				t.Errorf("got %v, want %s", result, tt.result)
			}
		})

	}
}
