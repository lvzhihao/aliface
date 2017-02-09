$( document ).ready(function(){
    wx.ready(function(){
        wx.chooseImage({
            count: 1, // 默认9
            sizeType: ['compressed'], // 可以指定是原图还是压缩图，默认二者都有
            sourceType: ['album', 'camera'], // 可以指定来源是相册还是相机，默认二者都有
            success: function (res) {
                document.getElementById('image').src = res.localIds[0]
                wx.uploadImage({
                    localId:  res.localIds[0],  // 需要上传的图片的本地ID，由chooseImage接口获得
                    isShowProgressTips: 1,      // 默认为1，显示进度提示
                    success: function (res) {
                        var serverId = res.serverId; // 返回图片的服务器端ID
                        $.ajax({
                            url: "/api/faceplusplus",
                            method: "POST",
                            contentType: "application/json; charset=UTF-8",
                            data: JSON.stringify({
                                "serverId":  res.serverId 
                            })
                        }).done(function(msg){
                            console.log(msg)
                                if(msg.error){
                                    alert("这张是人像吗？")
                                        return
                                }
                            data = msg.data
                                /*
                                   var imgNode = result.find('img, canvas')
                                   var img = imgNode[0]
                                   var pixelRatio = window.devicePixelRatio || 1
                                //console.log(img)
                                //console.log(pixelRatio)
                                imgNode.Jcrop({
                                setSelect: [
                                data.local.left / pixelRatio,
                                data.local.top / pixelRatio,
                                (data.local.left + data.local.width) / pixelRatio,
                                (data.local.top + data.local.height) / pixelRatio
                                ],
                                onSelect: function (coords) {
                                coordinates = coords
                                },
                                onRelease: function () {
                                coordinates = null
                                }
                                })
                                */
                                if( data.score > 0 ){
                                    alert("你好," + data.name)
                                }else {
                                    alert("系统需要和你交个朋友")
                                }
                        })


                    }
                });

                //var localIds = res.localIds; // 返回选定照片的本地ID列表，localId可以作为img标签的src属性显示图片
                //
            }
        });
    });
});
