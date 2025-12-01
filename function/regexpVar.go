package function

import "regexp"

// 样式正则
var styleRex = regexp.MustCompile(`<style[\s\S]*?<\/style>`)

// js脚本正则
var scriptRex = regexp.MustCompile(`<script[\s\S]*?<\/script>`)

// 汉字匹配正则
var hzRex = regexp.MustCompile("^[\u4e00-\u9fa5]$")

// 疑似入侵正则
var dangerRex = regexp.MustCompile(`/\\*(?:.|[\\n\\r])*\\*/`)

// 提取合格字符
var safeRex, _ = regexp.Compile("^[a-zA-Z0-9\u4e00-\u9fa5\\{1F300}-\\x{1F64F}\\x{1F680}-\\x{1F6FF}\\x{2600}-\\x{2B55},.!?:，。！？：<>《》/\"'.@= #`$%^&*()_+-、（）]+$")
