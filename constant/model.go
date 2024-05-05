package constant

type ErrMessage struct {
	Err     error // 错误
	ErrCode int   // 错误码
}

// MysqlProxy mysql的proxy
type MysqlProxy struct {
	Account  string // 账号
	Password string // 密码
	Host     string // IP
	Port     string // 端口
	DBName   string // mysql名称
}

// PageSizeReq 分页请求
type PageSizeReq struct {
	Page  int `json:"page"`  // 页
	Limit int `json:"limit"` // 页大小
}
