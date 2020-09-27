package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "io/ioutil"
    "mime/multipart"
    "net/http"
    "os"
    "strconv"
    "strings"
)

type LoginField struct {
    Username  string `json:"username"`
    Password  string `json:"password"`
    Recaptcha string `json:"recaptcha"`
}

func UnescapeUnicode(raw []byte) ([]byte, error) {
    str, err := strconv.Unquote(strings.ReplaceAll(strconv.Quote(string(raw)), `\\u`, `\u`))
    if err != nil {
        return nil, err
    }
    return []byte(str), nil
}

func login(username string, password string) (int, string) {
    b, err := json.Marshal(LoginField{"wedo", "wedo", ""})
    if err != nil {
        log.Errorf("json encode failed")
        return http.StatusBadRequest, ""
    }
    url := host + "/api/login"
    resp, err := http.Post(url, "application/json", bytes.NewBuffer(b))
    if err != nil {
        log.Errorf("http request post: %v", err)
        return http.StatusNotFound, ""
    }
    defer resp.Body.Close()

    respBody, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        log.Errorf("read response body failed: %v", err)
        return http.StatusNoContent, ""
    }

    respBody, _ = UnescapeUnicode(respBody)
    return resp.StatusCode, string(respBody)
}

func renew(jwt string) (int, string) {
    url := host + "/api/renew"
    req, err := http.NewRequest("POST", url, &bytes.Buffer{})
    if err != nil {
        log.Errorf("http new request failed: %v", err)
        return http.StatusNotFound, ""
    }
    req.Header.Set("X-Auth", jwt)
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        log.Errorf("http post request failed: %v", err)
        return http.StatusNotFound, ""
    }
    defer resp.Body.Close()

    respBody, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        log.Errorf("read response body failed: %v", err)
        return http.StatusNoContent, ""
    }

    respBody, _ = UnescapeUnicode(respBody)
    return resp.StatusCode, string(respBody)
}

func upload(path string, dir string, uuid string, jwt string) (int, string) {
    url := host + "/api/resources/wedo/"
    if strings.Contains(path, "ClientConfig") {
        idx := strings.Index(path, "ClientConfig")
        relative := path[idx:]
        query := fmt.Sprintf("?override=%s&dir=%s&uuid=%s", "true", dir, uuid)
        url += relative + query
    } else if strings.Contains(path, "ServerConfig") {
        idx := strings.Index(path, "ServerConfig")
        relative := path[idx:]
        query := fmt.Sprintf("?override=%s&dir=%s&uuid=%s", "true", dir, uuid)
        url += relative + query
    }

    bodyBuf := &bytes.Buffer{}
    bodyWriter := multipart.NewWriter(bodyBuf)
    fileWriter, err := bodyWriter.CreateFormFile("uploadfile", path)
    if err != nil {
        log.Errorf("write to buffer failed: %v", err)
        return http.StatusBadRequest, ""
    }

    f, err := os.Open(path)
    if err != nil {
        log.Errorf("open file failed: %v", err)
        return http.StatusBadRequest, ""
    }
    defer f.Close()

    _, err = io.Copy(fileWriter, f)
    if err != nil {
        log.Errorf("copy file content to body buffer failed: %v", err)
        return http.StatusBadRequest, ""
    }

    contentType := bodyWriter.FormDataContentType()
    bodyWriter.Close()

    req, err := http.NewRequest("POST", url, bodyBuf)
    if err != nil {
        log.Errorf("http new request failed: %v", err)
        return http.StatusNotFound, ""
    }
    req.Header.Set("X-Auth", jwt)
    req.Header.Set("Content-Type", contentType)

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        log.Errorf("http post request failed: %v", err)
        return http.StatusNotFound, ""
    }
    defer resp.Body.Close()

    respBody, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        log.Errorf("read response body failed: %v", err)
        return http.StatusNoContent, ""
    }

    respBody, _ = UnescapeUnicode(respBody)
    return resp.StatusCode, string(respBody)
}

func reloadConfig(uuid string, jwt string) (int, string) {
    url := host + "/api/reload?uuid=" + uuid

    reqest, err := http.NewRequest("GET", url, nil)
    if err != nil {
        log.Errorf("http new request failed: %v", err)
        return http.StatusNotFound, ""
    }
    reqest.Header.Add("X-Auth", jwt)

    client := &http.Client{}
    resp, err := client.Do(reqest)
    if err != nil {
        log.Errorf("http post request failed: %v", err)
        return http.StatusNotFound, ""
    }
    defer resp.Body.Close()

    respBody, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        log.Errorf("read response body failed: %v", err)
        return http.StatusNoContent, ""
    }

    respBody, _ = UnescapeUnicode(respBody)
    return resp.StatusCode, string(respBody)
}
