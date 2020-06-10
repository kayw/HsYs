Hs Ys crawler
====
海关商品编码申报要素获取

  **no warranty guaranteed**
本工具旨在测试或者学习用途，请勿用于非法用途或商业牟利。
若本工具被用于非法用途，造成一切后果与本作者无关。

## 功能:
  [x] hscode.net 申报要素列表抓取生成商品编码xlsx
  [x] 判断更新日志 未更新时不抓
  [] xlsx生成后发送邮件
  [] 抓取发生错误时告警
  [] 结构化代码 回调函数放到struct里面


## 使用:
   go v1.4+

   git clone
   go mod download
   go build -o hsys .

## License
  
  All packages are distributed under the MIT license. See the license [here](https://github.com/kayw/HsYs/blob/master/LICENSE).
