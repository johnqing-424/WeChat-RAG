# 微信-RAGFlow集成网络问题解决方案

## 问题描述

在微信公众号与RAGFlow集成过程中，我们遇到了以下关键问题：

1. **微信无法接收RAGFlow回答**：尽管RAGFlow能正常处理问题并生成回答，但微信端无法收到这些回答。
2. **会话管理问题**：不同用户的问题可能混淆，导致回答不一致。
3. **响应超时问题**：微信要求5秒内响应，但RAGFlow生成答案需要更长时间。

## 排查过程

1. **超时处理优化**：
   - 将HTTP请求超时从30秒增加到120秒
   - 将Context超时从35秒增加到140秒
   - 实现了基于消息ID的缓存机制，支持超时后异步完成回答

2. **网络连接问题排查**：
   - 发现RagFlow服务URL使用了外部IP地址（http://114.215.255.105:8081）
   - 经测试，从主机无法通过容器名称（ragflow-server）直接访问RAGFlow服务
   - 确认RAGFlow服务在Docker网络"docker_ragflow"中运行，但WeChat-RAG应用未加入此网络

3. **消息格式问题**：
   - 发现XML响应格式不符合微信标准（使用了`<WeChatResponse>`而非`<xml>`作为根元素）
   - 响应内容中没有使用CDATA包装文本内容
   - 系统原本删除了RAGFlow返回的`##$$`标记，而这是显示引用原文的重要标记

## 解决方案

1. **修改服务URL配置**：
   - 将RagFlow服务URL从IP地址改为Docker容器名称（http://ragflow-server）
   ```go
   // 修改前
   const RagFlowBaseURL = "http://114.215.255.105:8081"
   // 修改后
   const RagFlowBaseURL = "http://ragflow-server"
   ```

2. **建立网络连接**：
   - 创建网络连接脚本（fix_network.sh）执行以下步骤：
     1. 创建一个网络代理容器，加入RAGFlow的Docker网络
     2. 获取RAGFlow服务的内部IP地址
     3. 将RAGFlow服务的IP和主机名添加到主机的/etc/hosts文件
     4. 重启WeChat-RAG服务以应用新配置

3. **修正XML消息格式**：
   - 创建专门的XML响应生成函数，确保格式符合微信标准：
   ```go
   func createWeChatXMLResponse(fromUser, toUser, content string) string {
       timestamp := time.Now().Unix()
       xmlFormat := `<xml>
   <ToUserName><![CDATA[%s]]></ToUserName>
   <FromUserName><![CDATA[%s]]></FromUserName>
   <CreateTime>%d</CreateTime>
   <MsgType><![CDATA[text]]></MsgType>
   <Content><![CDATA[%s]]></Content>
   </xml>`
       return fmt.Sprintf(xmlFormat, toUser, fromUser, timestamp, content)
   }
   ```
   - 修改清理函数，保留有用的标记：
   ```go
   func cleanAnswer(answer string) string {
       // 只去除不必要的标记，保留原文引用标记##$$
       specialMarks := []string{"CITATIONS:", "CITATIONS: "}
       for _, mark := range specialMarks {
           answer = strings.Replace(answer, mark, "", -1)
       }
       return strings.TrimSpace(answer)
   }
   ```
   - 使用c.String()而非c.XML()发送响应，避免Gin框架自动添加XML封装

4. **测试验证**：
   - 通过测试脚本验证不同用户可以获得正确回答
   - 验证响应格式符合微信要求
   - 保留了RAGFlow原文引用标记`##$$`用于显示引用来源

## 总结

本次问题修复涉及三个关键方面：网络连接、超时处理和消息格式。通过Docker网络配置解决了服务间通信问题，通过异步处理和缓存解决了超时问题，通过自定义XML格式解决了微信消息标准不匹配的问题。

现在系统能够在微信要求的时间内返回初始响应，并异步获取完整答案，同时保留RAGFlow提供的原文引用标记，以便用户查看信息来源。

## 建议

1. **集成部署**：最好将微信服务也容器化，并加入同一个Docker Compose配置中
2. **网络配置**：确保在docker-compose.yml中正确设置网络配置
3. **监控工具**：添加网络连接和响应格式监控，及时发现类似问题
4. **微信协议**：严格遵循微信XML消息格式标准，保证内容正确包装

## 参考资料

1. Docker网络配置：https://docs.docker.com/network/
2. 微信公众号开发指南：https://open.weixin.qq.com/
3. 微信消息格式规范：https://developers.weixin.qq.com/doc/offiaccount/Message_Management/Receiving_standard_messages.html
4. RAGFlow API文档：https://ragflow.io/docs/dev/ 