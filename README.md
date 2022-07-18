# grpc
1.serialize
the frame use gob as a serialize

client ---- http message ---- server

http: head
      body
rpc :head
     body

/**
| Option{MagicNumber: xxx, CodecType: xxx} | Header{ServiceMethod ...} | Body interface{} |
| <------      固定 JSON 编码      ------>  | <-------   编码方式由 CodeType 决定   ------->|
**/

