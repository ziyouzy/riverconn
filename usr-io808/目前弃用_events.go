//目前看来events的分支结构可以用来处理析构问题，以及诸如notpasscrc.go这样的根据业务需求临时出现的业务逻辑
//目前看来该swtich应该属于更上层逻辑
//因为在初始化一个Client时，内部的Singals和Errors字段也是外部赋予的引用地址，而不是创建的本体
func (p *Client)Events(){

}