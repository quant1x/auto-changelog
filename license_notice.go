package main

import _ "embed"

// noticesText 嵌入第三方依赖许可证声明（由 go-licenses 生成，见 third_party/NOTICE.txt）。
// 通过 --license 参数输出，保障单一可执行文件随自身携带合规的许可证信息。
//
// 重新生成：
//
//	go run github.com/google/go-licenses@latest report ./... \
//	  --template third_party/notice.tmpl \
//	  --ignore gitee.com/quant1x/autochangelog > third_party/NOTICE.txt
//
//go:embed third_party/NOTICE.txt
var noticesText string
