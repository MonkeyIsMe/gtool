package constant

// 返回1xxxx -> 请求成功
const (
	Success           = 10000 // 请求成功的时候的返回值
	DataEmpty         = 10001 // 请求的数据为空时候的返回值
	ParameterMiss     = 10002 // 请求所需的参数部分缺失
	ParameterInvalid  = 10003 // 请求的参数存在不合法的情况
	LoginExpired      = 10004 // 登录失效
	InfoNotMatch      = 10005 // 信息不匹配
	PrivilegeNotMatch = 10006 // 权限不匹配
)

// 返回2xxxx -> MySQL错误
const (
	QueryError       = 20000 // MySQL的查询错误
	ScanError        = 20001 // MySQL在Scan时候的错误
	InsertError      = 20002 // MySQL的插入错误
	DeleteError      = 20003 // MySQL的删除错误
	UpdateError      = 20004 // MySQL的更新错误
	CountError       = 20005 // MySQL的计数错误
	AffectRowError   = 20006 // MySQL获取影响行数错误
	PrepareError     = 20007 // MySQL 插入时Prepare的错误
	ExecError        = 20008 // MySQL 插入时Exec的错误
	TransactionError = 20009 // MySQL 事务操作时的错误
	GetIDError       = 20010 // 获取ID错误
)

// 返回3xxxx -> 其他类型的错误
const (
	StrconvAtoiError = 30000 // 类型转换错误（主要是string转int）
	ParseTimeError   = 30001 // 日期类型的错误
	UnknownError     = 30002 // 未知错误
	ModelIsExist     = 30003 // 该类型已存在
	CacheError       = 30004 // 缓存相关错误
)

// 返回4xxxx -> HTTP错误
const (
	HTTPDoReqError  = 40000 // http请求doRequest错误
	HTTPNewReqError = 40001 // http请求new Request错误
	HTTPGetError    = 40002 // http请求get错误
	HTTPPostError   = 40003 // http请求post错误
)

// 返回5xxxx -> json错误
const (
	JSONUnmarshalError = 50000 // json反序列化错误
	JSONMarshalError   = 50001 // json序列化错误
)

// 返回6xxxx -> 文件操作错误
const (
	ReadFileError   = 60000 // 读取文件错误
	FileIsNotExist  = 60001 // 文件不存在
	OpenFileError   = 60002 // 打开文件错误
	WriteFileError  = 60003 // 文件写入错误
	RemoveFileError = 60004 // 删除文件错误
	CreateFileError = 60005 // 创建文件错误
)
