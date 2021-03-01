/** 需要允许系统可以同时处理多个usr_io808设备，因此他的上层会存在如下形式:
 * map[[]byte]*usr_io808.Client从而实现“管理者”的相关操作
 * 需要让他体现出是属于tcpsocket，udpsocket，serialsocket，snmpsocket中的哪一种
 * 这是添加stamps所必要的内容

 ** Client.Singals与Client.Errors的本体并不会在此包创建，而是在此包的上层创建
 * 上层，此层(当前包).下层(heartbeating,crc,stamps等)会公用同一Signals/Errors
 */
package usr_io808

import (
    "github.com/ziyouzy/river-node"
    "github.com/ziyouzy/river-node/heartbreating"
    "github.com/ziyouzy/river-node/crc"
    "github.com/ziyouzy/river-node/stamps"

    "net"
    "strconv"
    "crypto/md5"
    "encoding/hex"
)


type Client struct{
    PlainUniqueId string
	Singals chan river_node.Singal 
    Errors chan error

    sources net.Conn//数据源

    riverNodes []*river_node.RiverNode

    
    /*如下可以直观的体现出需要加载哪些river-node*/
    hbRaws chan struct{}
    heartBeatingConfig *heartbeating.HeartBeatingConfig
    
    crcRaws chan []byte;    crcPassNews chan []byte;    crcNotPassNews chan []byte
    crcConfig *crc.CRCConfig

    stampsRaws chan []byte;    Fin_StampsNews chan []byte//返回值
    stampsConfig *stamps.StampsConfig 
}


/*attach方法会在检查完Singals与Errors是否可以之后才被调用，因此可以直接使用这两个管道*/
func (p *Client) attach(riverNodeName string, config Config)bool{
	for _, riverNode := range p.riverNodes {
		if riverNode.Name == riverNodeName {
            p.Errors <- river_node.NewError(RIVERCONN_USRIO808_INITFAIL, p.PlainUniqueId, 
                                  fmt.Sprintf("river-node:%s已被装载与当前riverconn,"+
                                     "导致了uid为%s的usr-io8080设备初始化失败", 
                                     riverNodeName, p.PlainUniqueId))
            return false
		}
    }
    
	riverNodeFun, ok := river_node.RegisteredNodes[riverNodeName]
	if !ok {
        p.Errors <- river_node.NewError(RIVERCONN_USRIO808_INITFAIL, p.PlainUniqueId, 
                              fmt.Sprintf("river-node:%s并不在册,导致了uid为%s"+
                                 "的usr-io8080设备初始化失败", riverNodeName, p.PlainUniqueId))
        return false
    }
    
	riverNodeAbs := riverNodeFun()
    err := riverNodeAbs.Init(config)
    
	if err != nil {
        p.Errors <- river_node.NewError(RIVERCONN_USRIO808_INITFAIL, p.PlainUniqueId, 
                              fmt.Sprintf("river-node:%s适配器在初始化时发生了如下错误[%s],"+
                                 "导致了uid为%s的usr-io8080设备初始化失败",
                                 riverNodeName, err.Error(), p.PlainUniqueId))
        return false
	}

	riverNode := &river_node.RiverNode{
		Name:           riverNodeName,
		NodeAbstract:   riverNodeAbs,
	}

    p.riverNodes = append(p.riverNodes, riverNode)

    return true
}



/** plainUniqueId的格式为：
 * 诸如"192.168.1.10:6668:TCP"
 * 诸如"192.168.1.11:6669:UDP"
 * 诸如"tty1:NULL:SERIAL"
 * 诸如"192.168.1.13:300:SNMP"

 ** Init()失败等同于本次客户端连接失败,然而整个程序并不会因此直接panic
   只负责装配各适配器、进行各适配器的传参，赋值，初始化
   只负责连接管道，不负责注入管道
 */
func (p *Client)Init(sources net.Conn) err error{
    if p.Signals ==nil || p.Errors ==nil || p.PlainUniqueId ==""{
        err = errors.New("one usr-io808 client init error, "+
                    "Signals or Errors PlainUniqueId is nil")
        return
    }
 
    if sources ==nil{
        err = errors.New("one usr-io808 client init error, Sources is nil")
        return
    }else{
        p.sources =sources
    }
    

    //if plainUniqueId//检查格式

//心跳包初始化
    p.hbRaws =make(chan struct{})
    p.heartBeatingConfig   = &heartbeating.HeartBeatingConfig{
        UniqueId:       fmt.Sprintf("%s:%s:%s",p.PlainUniqueId,p.con.RemoteAddr().String(),
                           "heartbeating"),

        Signals:        p.signals,
        Errors:         p.errors,
 
        TimeoutSec:     8 * time.Second,//后期可以直接读取配置文档
        TimeoutLimit:   3,//后期可以直接读取配置文档

        Raws:           p.hbRaws,
    }
    if ok := p.attach(heartbeating.RIVER_NODE_NAME, heartBeatingConfig); !ok{
        err = errors.New("one usr-io808 client init error, attach heartbeating fail")
        return
    }
//心跳包初始化结束

//crc初始化
    p.crcRaws =make(chan []byte)
    p.crcPassNews = make(chan []byte)
    p.crcNotPassNews = make(chan []byte)
    p.crcConfig   = &crc.CRCConfig{
        UniqueId:      fmt.Sprintf("%s:%s:%s",p.PlainUniqueId,p.con.RemoteAddr().String(),
                          "crc"),

        Signals:       p.signals, 
        Errors:        p.errors,

        Mode:          crc.NEWCHAN,//后期可以直接读取配置文档
        IsBigEndian:   crc.ISBIGENDDIAN,//后期可以直接读取配置文档
        NotPassLimit:  20,//后期可以直接读取配置文档

        Raws:          p.crcRaws, 
     
        PassNews:      p.crcPassNews, 
        NotPassNews:   p.crcNotPassNews, 
    }
    if ok := p.attach(crc.RIVER_NODE_NAME, crcConfig); !ok{
        err = errors.New("one usr-io808 client init error, attach crc fail")
        return
    }
//crc初始化结束

//stamps初始化
    p.stampsRaws =make(chan []byte)
    p.StampsNews =make(chan []byte)
    p.stampsConfig = &stamps.StampsConfig{
       
        UniqueId:        fmt.Sprintf("%s:%s:%s",p.PlainUniqueId,p.con.RemoteAddr().String(),
                            "stamps"),

        Signals:         p.Signals,
        Errors:          p.Errors,
 
        Mode:            stamps.HEAD, //后期可以直接读取配置文档
        AutoTimeStamp:   true,
        Breaking:        []byte("/-/"), //后期可以直接读取配置文档
        //在旧代码中，tag :=[]byte("tcpsocket")
        Stamps:          []byte(p.PlainUniqueId)) //后期可以直接读取配置文档          
        
        Raws:            p.stampsRaws,
     
        News:            p.Fin_StampsNews,
    }
    
    if ok := p.attach(stamps.RIVER_NODE_NAME, stampsConfig); !ok{
        err = errors.New("one usr-io808 client init error, attach stamps fail")
        return
    }
    
//stamps初始化结束

//对各个river-node进行连接的其他必要操作
    go func(){
        for res := range CRCNotPassNews{
            //最终的结果都只会是转化为某种signal或者error
            //不过这些未通过的数据本身不会有太大的使用价值
            if len(res) ==4 && res[0] ==0x49 && res[1] == 0x4f{
                p.Signals <- river_node.NewSignal(RIVERCONN_USRIO808_NEWUSRIO808,
                                       p.PlainUniqueId,string(res))
            }else{
                p.Errors  <- river_node.NewError(RIVERCONN_USRIO808_RESOLVESRCFAIL,
                                       p.PlainUniqueId,hex.EncodeToString(res))
            }	
        }
    }()

    go func(){
        for res := range CRCPassNews{
            p.config.stampsRaws <- res	
        }
    }()
//对各个river-node进行连接的其他必要操作完成


    signal_success := river_node.NewSignal(RIVERCONN_USRIO808_INITSUCCESS, p.PlainUniqueId, 
                                "客户端类型为某usr-io808终端")
    p.Signals <- signal_success
    return
}


//Run()负责管道注入
func (p *Client)Run(){
    p.Signals <- river_node.NewSignal(RIVERCONN_USRIO808_RUN, p.PlainUniqueId, "")
    go func(){
        reSpoon := make([]byte, 4096);    resLen int
        /** 循环内部不会涉及任何break或return操作
         * 可使循环结束的途径只有:
         * heartbeating.HEARTBREATING_PANIC信号和
         * heartbeating.CRC_PANIC信号
         * 他们不仅仅会跳出循环，也会直接析构掉当前usr-io808客户对象
         */
	    for {
		    resLen, err := p.sources.Read(reSpoon)
		    if (err != nil)&&(err !=io.EOF) 
                p.Errors<-river_node.NewError(RIVERCONN_USRIO808_READCONNERR,p.PlainUniqueId,
                                    fmt.Sprintf("原生的错误内容:%s",err))
                continue
            }	

            if err == io.EOF {
                p.hbRaws <- struct{}{}
                continue
            }

            res :=reSpoon[:resLen]
            hbRaws <- struct{}{}
            p.crcRaws <- res
        }
    }()
}