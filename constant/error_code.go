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
	CountError       = 20004 // MySQL的计数错误
	AffectRowError   = 20005 // MySQL获取影响行数错误
	PrepareError     = 20006 // MySQL 插入时Prepare的错误
	ExecError        = 20007 // MySQL 插入时Exec的错误
	TransactionError = 20007 // MySQL 事务操作时的错误
	GetIDError       = 20008 // 获取ID错误
)

// 返回3xxxx -> HTTP错误
const (
	HTTPDoReqError  = 30000
	HTTPNewReqError = 30001
	HTTPGetError    = 30002
	HTTPPostError   = 30003
)

// 返回4xxxx -> json错误
const (
	JSONUnmarshalError = 40000 // json反序列化错误
	JSONMarshalError   = 40001 // json序列化错误
)

// 返回5xxxx -> 文件操作错误
const (
	ReadFileError   = 50000 // 读取文件错误
	FileIsNotExist  = 50001 // 文件不存在
	OpenFileError   = 50002 // 打开文件错误
	WriteFileError  = 50003 // 文件写入错误
	RemoveFileError = 50004 // 删除文件错误
	CreateFileError = 50005 // 创建文件错误
)

// 返回9xxxx -> 其他类型的错误
const (
	StrconvAtoiError = 90000 // 类型转换错误（主要是string转int）
	ParseTimeError   = 90001 // 日期类型的错误
	UnknownError     = 90002 // 未知错误
	ModelIsExist     = 90003 // 该类型已存在
	CacheError       = 90004 // 缓存相关错误

)
