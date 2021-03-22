package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"image"
	"log"
	"net"
	"os"
	"pin2pre/cacheFile"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type display struct {
	Product []string `json:"Product"`
}

var mp map[int]string = make(map[int]string)
var cacheObject cacheFile.Cache = cacheFile.NewCache()

type data struct {
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
	Price    int    `json:"price"`
}

var (
	db          *sql.DB
	q           int
	newQuantity int
	mutex       sync.Mutex
)

type respond struct {
	Msg string `json:"msg"`
}

// var count int = 0

func main() {
	db, _ = sql.Open("mysql", "root:62011139@tcp(127.0.0.1:3306)/prodj")
	li, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalln(err.Error())
	}
	defer li.Close()
	for {
		conn, err := li.Accept()

		if err != nil {
			log.Fatalln(err.Error())
			continue
		}
		go handle(conn)
	}
}

func handle(conn net.Conn) {
	defer conn.Close()
	req(conn)

}

func req(conn net.Conn) {
	buffer := make([]byte, 1024)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		message := string(buffer[:n])
		if !strings.Contains(message, "HTTP") {
			if _, err := conn.Write([]byte("Recieved\n")); err != nil {
				log.Printf("failed to respond to client: %v\n", err)
			}
			break
		}
		headers := strings.Split(message, "\n")
		method := (strings.Split(headers[0], " "))[0]
		path := (strings.Split(headers[0], " "))[1]
		p := strings.Split(path, "/")

		if p[1] == "" {
			home(conn, method, "pre-order/index.html", "text/html")
			break
		} else if p[1] == "products" {
			if (len(p) > 2) && (p[2] != "") {
				fmt.Println("message", message)
				result := getJson(message)
				// fmt.Println(result)
				productWithID(conn, method, p[2], result)
				break
			} else {
				fmt.Println("HI")
				products(conn, method)
				break
			}
		} else if p[1] == "style.css" {
			home(conn, method, "pre-order/style.css", "text/css")
			break
		} else if p[1] == "images" {
			f := p[2]
			nf := "pre-order/images/" + f
			homeImg(conn, method, nf, "image/apng")
			break
		}
	}

}

func getJson(message string) data {
	var result data
	if strings.ContainsAny(string(message), "}") {

		r, _ := regexp.Compile("{([^)]+)}")
		match := r.FindString(message)
		fmt.Println(match)
		fmt.Printf("%T\n", match)
		json.Unmarshal([]byte(match), &result)
		fmt.Println("data", result)
	}
	return result
}

func homeImg(conn net.Conn, method string, filename string, t string) {
	if method == "GET" {
		c := t
		d, _ := getImageFromFilePath(filename)
		sendFile(conn, d, c)
	}
}

func getImageFromFilePath(filePath string) (image.Image, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	image, _, err := image.Decode(f)
	return image, err
}

func home(conn net.Conn, method string, filename string, t string) {
	if method == "GET" {
		c := t
		d := call_cache(filename)
		send(conn, d, c)
	}
}

func products(conn net.Conn, method string) {
	if method == "GET" {
		d := display_pro()
		c := "application/json"
		send(conn, d, c)
	}
}

func productWithID(conn net.Conn, method string, id string, result data) {
	fmt.Println("ID")
	i, _ := strconv.Atoi(id)
	if method == "GET" {
		mutex.Lock()
		d := cache(i)
		mutex.Unlock()
		c := "application/json"
		send(conn, d, c)
	} else if method == "POST" {
		fmt.Println("here")
		fmt.Println(result.Quantity)
		success := postPreorder(i, result.Quantity)
		msg := ""
		if success == true {
			msg = "success"
		} else {
			msg = "fail"
		}
		jsonStr := respond{Msg: msg}
		jsonData, err := json.Marshal(jsonStr)
		if err != nil {
			fmt.Println("error post", err)
		}
		d := string(jsonData)
		c := "application/json"
		send(conn, d, c)
	}

}

func call_cache(filename string) string {
	start := time.Now()
	d, err := cacheObject.Check(filename)
	if err != nil {
		fmt.Println(err)
		a := getFile(filename)
		cacheObject.Add(filename, a)
		d, _ = cacheObject.Check(filename)
		cacheObject.Display()

		fmt.Println("Time calling cache miss: ", time.Since(start))
		return d
	} else {
		cacheObject.Display()

		fmt.Println("Time calling cache hit: ", time.Since(start))
		return d
	}

}

func getFile(filename string) string {
	// call_cache(filename)
	start := time.Now()
	f, err := os.Open(filename)
	if err != nil {
		fmt.Println("File reading error", err)

	}
	defer func() {
		if err := f.Close(); err != nil {
			panic(err)
		}
	}()
	chunksize := 512
	reader := bufio.NewReader(f)
	part := make([]byte, chunksize)
	buffer := bytes.NewBuffer(make([]byte, 0))
	var bufferLen int
	for {
		count, err := reader.Read(part)
		if err != nil {
			break
		}
		bufferLen += count
		buffer.Write(part[:count])
	}
	// fmt.Println("home")
	fmt.Println("Time get file: ", time.Since(start))
	return buffer.String()
	// contentType = "text/html"
	// headers = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Length: %d\r\nContent-Type: %s\r\n\n%s", bufferLen, contentType, buffer)

}

func sendFile(conn net.Conn, d image.Image, c string) {
	fmt.Fprintf(conn, createHeaderFile(d, c))
}

func createHeaderFile(d image.Image, contentType string) string {

	contentLength := 0

	headers := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Length: %d\r\nContent-Type: %s\r\n\n%s", contentLength, contentType, d)
	// fmt.Println(headers)
	return headers
}

func send(conn net.Conn, d string, c string) {
	fmt.Fprintf(conn, createHeader(d, c))
}

//create header function
func createHeader(d string, contentType string) string {
	contentLength := len(d)
	headers := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Length: %d\r\nContent-Type: %s\r\n\n%s", contentLength, contentType, d)
	return headers
}

func checkErr(err error) {
	if err != nil {
		fmt.Println("check err", err)
	}
}

func cache(id int) string {
	if val, ok := mp[id]; ok {
		fmt.Println("----------HIT----------")
		return val
	} else {
		return db_query(id)
	}
}

func db_query(id int) string {
	start := time.Now()
	// db, err := sql.Open("mysql", "root:62011139@tcp(127.0.0.1:3306)/prodj")
	// checkErr(err)

	fmt.Println("----------MISS----------")

	rows, err := db.Query("SELECT name, quantity_in_stock, unit_price FROM products WHERE product_id = " + strconv.Itoa(id))
	checkErr(err)

	for rows.Next() {
		var name string
		var quantity int
		var price int
		err = rows.Scan(&name, &quantity, &price)

		result := data{Name: name, Quantity: quantity, Price: price}
		byteArray, err := json.Marshal(result)
		checkErr(err)

		mp[id] = string(byteArray)

	}
	rows.Close()
	val := mp[id]
	fmt.Printf("time query from db: %v\n", time.Since(start))
	return val
}

func display_pro() (val string) {
	var l []string
	for i := 1; i <= 10; i++ {
		val := db_query(i)
		l = append(l, val)
	}

	result := display{Product: l}

	byteArray, err := json.Marshal(result)
	checkErr(err)

	val = string(byteArray)
	fmt.Println(val)
	return
}

func getQuantity(t chan int, id int) {
	start := time.Now()
	info := cache(id)

	var quan data
	err := json.Unmarshal([]byte(info), &quan)
	checkErr(err)
	t <- quan.Quantity

	fmt.Printf("time query from cache: %v\n", time.Since(start))
	fmt.Println("Quantity: ", quan.Quantity)

}

func decrement(t chan int, transactionC chan bool, orderQuantity int, id int) {
	start := time.Now()
	quantity := <-t // channel from getQuantity
	newQuantity := quantity - orderQuantity
	if newQuantity < 0 {
		transactionC <- false
		return
	}
	fmt.Println("New Quantity: ", newQuantity)
	db.Exec("update products set quantity_in_stock = ? where product_id = ? ", newQuantity, id)

	if _, ok := mp[id]; ok {
		info := cache(id)
		var quan data
		err := json.Unmarshal([]byte(info), &quan)

		result := data{Name: quan.Name, Quantity: newQuantity, Price: quan.Price}
		byteArray, err := json.Marshal(result)
		checkErr(err)
		mp[id] = string(byteArray)
		fmt.Println("punepit eiei", mp[id])
	}

	transactionC <- true
	fmt.Printf("time decrement: %v\n", time.Since(start))
	fmt.Printf(mp[id])
	return

}

func insert(user string, id int, q int) {
	start := time.Now()
	db.Exec("INSERT INTO order_items(username, product_id, quantity) VALUES (?, ?, ?)", user, id, q)
	fmt.Printf("time insert: %v\n", time.Since(start))
}

func preorder(end chan bool, user string, productId int, orderQuantity int) bool {
	// fmt.Printf("start\n")
	start := time.Now()
	transactionC := make(chan bool)
	t := make(chan int)
	// var success bool
	mutex.Lock()
	go getQuantity(t, productId)
	go decrement(t, transactionC, orderQuantity, productId)
	success := <-transactionC // wait for all go routines
	mutex.Unlock()
	if success {
		go insert(user, productId, orderQuantity)
	}
	fmt.Printf("\ntime: %v\n", time.Since(start))
	end <- success
	return success
}

func postPreorder(id int, quantity int) bool {
	//db, _ = sql.Open("mysql", "root:62011212@tcp(127.0.0.1:3306)/prodj")
	//defer db.Close()
	// n := 100 //
	end := make(chan bool) //, n)
	start2 := time.Now()

	go preorder(end, "1", id, quantity)

	success := <-end

	fmt.Printf("Total time: %v\n", time.Since(start2))
	fmt.Println("---------------")

	return success
}
