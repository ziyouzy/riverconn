package usr_io808

import (

	"net"
)


type Client struct{
	singals chan river-node.Singal 
	errors chan error

	con net.Conn

	hb river-node.HeartBreating
	crc river-node.crc
	stamp river-node.stamp
}

func (p *Client)Init(){

}

func (p *Client)Events(){

}

func (p *Client)crcConnectstamp(){
    TestDataCreaterNews = make(chan []byte)
    HBRaws              = make(chan struct{})
    CRCRaws             = make(chan []byte)
	go func(){
        for bytes := range TestDataCreaterNews{
            fmt.Println("source bytes is", bytes)
            HBRaws <- struct{}{}
            CRCRaws <- bytes	
        }
    }()
}

func (p *Client)ConnectTo(p chan physicalnode){
	
}