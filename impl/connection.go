package impl

import (
	"errors"
	"github.com/gorilla/websocket"
	"sync"
)

type Connection struct{
	wsConn *websocket.Conn
	inChan chan[]byte
	outChan chan[]byte
	closeChan chan byte
	mutex sync.Mutex
	isClosed bool

}
func InitConnect(wsConn *websocket.Conn)(conn *Connection, err error){
	conn = &Connection{
		wsConn: wsConn,
		inChan: make(chan []byte, 1000),//如果满了就无法放进去
		outChan: make(chan []byte, 1000),
		closeChan: make(chan byte, 1),
	}
	//启动读协程
	go conn.ReadLoop()
	//启动写协程
	go conn.WriteLoop()
	return
}
func (conn *Connection)ReadMessage()(data[]byte, err error){
	select {
	case data = <-conn.inChan:

	case <- conn.closeChan:
		err = errors.New("connection is closed")
	}
	return data, err
}
func (conn *Connection)WriteMessage(data[]byte)(err error){
	select {
	case conn.outChan <- data:

	case <- conn.closeChan:
		err = errors.New("connection is closed")
	}
	return err
}
func (conn *Connection) Close (){
	conn.mutex.Lock()
	if !conn.isClosed{
		//保证这一行代码只执行一次
		close(conn.closeChan)
		conn.isClosed = true
	}
	conn.mutex.Unlock()
	//线程安全的可重入的closed
	conn.wsConn.Close()

}
//内部实现
func (conn *Connection) ReadLoop (){
	var(
		data []byte
		err error
	)
	for{
		if _, data, err = conn.wsConn.ReadMessage(); err != nil{
			goto ERR
		}
		select {
		case conn.inChan <- data:
		case <- conn.closeChan:
			goto ERR
		}

	}
	//阻塞在这里，等待inchan有空闲位置
	ERR:
		conn.Close()
}
func (conn *Connection) WriteLoop (){
	var(
		data []byte
		err error
	)
	for{
		select {
		case data = <- conn.outChan:
		case <- conn.closeChan:
			goto ERR
		}

		if err = conn.wsConn.WriteMessage(websocket.TextMessage, data); err != nil{
			goto ERR
		}

	}
ERR:
	conn.Close()
}