// common connection pool to connect to another server
//caller asure to return the connection to the pool
package connpool

import (
	"container/list"
	"sync"
	"time"
    "errors"
    "fmt"
)

var NowFunc = time.Now // for testing

type ConnPool struct {
	Connect    func() (interface{}, int, error) //connect func,return instance,id,error
	DisConnect func(c interface{}, id int)      //disconnect func

	//must init param
	// Maximum number of connections allocated by the pool at a given time.
	// When zero, there is no limit on the number of connections in the pool.
	MaxActiveNum int

	//Reserved idle connections
	ReservedIdleNum int

	// Close connections after remaining idle for this duration. If the value
	// is zero, then idle connections are not closed. Applications should set
	// the timeout to a value less than the server's timeout.
	IdleTimeout time.Duration

	// How many seconds wait for when the pool is at the MaxActiveNum limit
	//0 forever,-1 no wait
	WaitTime int64

	mu     sync.RWMutex
    cond   *TMOCond
	closed bool

	idlePool list.List

	//internal param
	activeNum int //current inuse num
	waitNum   int //wait num
}

type Conn struct {
	t    time.Time   //time duration
	Err  error       // whether the conn is error
	Inst interface{} //conn instance
	ID   int         //the id of the conn,given by the connect func
}

// new connection pool
func NewConnectionPool(maxActiveNum int, revIdleNum int, idleTimeout time.Duration, waitTime int64, connectFunc func() (interface{}, int, error), disConnectFunc func(c interface{}, id int)) *ConnPool {
	return &ConnPool{
		MaxActiveNum:    maxActiveNum,
		ReservedIdleNum: revIdleNum,
		IdleTimeout:     idleTimeout,
		WaitTime:        waitTime,
		Connect:         connectFunc,
		DisConnect:      disConnectFunc,
	}
}

//pop an connection from pool
func (p *ConnPool) Pop() (*Conn, error) {
	var c *Conn
    tryed := false
	p.mu.Lock()

	//for loop to close idle timeout conn and close them
	if timeout := p.IdleTimeout; timeout > 0 {
		for i, n := p.ReservedIdleNum, p.idlePool.Len(); i < n; i++ {
			e := p.idlePool.Back()
			if e == nil {
				break
			}
			c := e.Value.(*Conn)
			if c.t.Add(timeout).After(NowFunc()) {
				break
			}
			p.idlePool.Remove(e)
			p.mu.Unlock()
			go p.DisConnect(c.Inst, c.ID)
			p.mu.Lock()
		}
	}

	for {
		//Get idle connection.
		if p.idlePool.Len() > 0 {
			e := p.idlePool.Front()
			c = e.Value.(*Conn)
			p.idlePool.Remove(e)

			// mark as in use
			p.activeNum += 1
			p.mu.Unlock()
			return c, nil
		}

		// Check for pool closed before dialing a new connection.
		if p.closed {
			p.mu.Unlock()
			return nil, errors.New("pool closed")
		}

		if p.MaxActiveNum == 0 || p.activeNum < p.MaxActiveNum {
			p.activeNum += 1
			p.mu.Unlock()
			//new connection
			Inst, id, e := p.Connect()
			if e != nil {
				p.mu.Lock()
				p.activeNum -= 1
				p.mu.Unlock()
				return nil, fmt.Errorf("connection pool:%s", e.Error())
			}

			// init struct
			c = &Conn{Inst: Inst, ID: id}
			return c, nil
		}

        //no wait
		if p.WaitTime < 0 {
			p.mu.Unlock()
			return nil, nil
		}

        if tryed {
            return nil, errors.New("pop wait timeout")
        }

        if p.cond == nil {
            p.cond = NewTMOCond(&p.mu)
        }

        p.waitNum += 1
        if p.WaitTime > 0 {
            p.cond.WaitOrTimeout(time.Second * time.Duration(p.WaitTime))
            tryed = true
        }else {
            p.cond.Wait()
        }
        p.waitNum -= 1
	}
}

//push an connection to pool.(you shoud not op c after push the conn to pool)
func (p *ConnPool) Push(c *Conn) error {
	if c == nil {
		return errors.New("connection pool:[Push] c == nil")
	}

	// if the conn is err,drop it
	if c.Err != nil {
		p.mu.Lock()
		p.activeNum -= 1
		p.mu.Unlock()
		p.DisConnect(c.Inst, c.ID)
		//return fmt.Errorf("connection pool:drop error connection,id:%d,err:%s", c.ID, c.Err.Error())
        return nil
    }

	c.t = NowFunc()

	p.mu.Lock()
	p.idlePool.PushFront(c)
	p.activeNum -= 1
	if p.cond != nil {
		p.cond.Signal()
	}

	p.mu.Unlock()
    return nil
}

func (p *ConnPool) GetActiveNum() int {
	p.mu.RLock()
	an := p.activeNum
	p.mu.RUnlock()
	return an
}

func (p *ConnPool) GetIdleNum() int {
	p.mu.RLock()
	in := p.idlePool.Len()
	p.mu.RUnlock()
	return in
}

func (p *ConnPool) GetWaitNum() int {
	p.mu.RLock()
	waitNum := p.waitNum
	p.mu.RUnlock()
	return waitNum
}
