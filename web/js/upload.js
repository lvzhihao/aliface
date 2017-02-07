/*
 * JavaScript Load Image Demo JS
 * https://github.com/blueimp/JavaScript-Load-Image
 *
 * Copyright 2013, Sebastian Tschan
 * https://blueimp.net
 *
 * Licensed under the MIT license:
 * http://www.opensource.org/licenses/MIT
 */

/* global loadImage, HTMLCanvasElement, $ */

$(function () {
  'use strict'

  var result = $('#result')
  var exifNode = $('#exif')
  var thumbNode = $('#thumbnail')
  var actionsNode = $('#actions')
  var currentFile
  var coordinates

  function displayExifData (exif) {
    var thumbnail = exif.get('Thumbnail')
    var tags = exif.getAll()
    var table = exifNode.find('table').empty()
    var row = $('<tr></tr>')
    var cell = $('<td></td>')
    var prop
    if (thumbnail) {
      thumbNode.empty()
      loadImage(thumbnail, function (img) {
        thumbNode.append(img).show()
      }, {orientation: exif.get('Orientation')})
    }
    for (prop in tags) {
      if (tags.hasOwnProperty(prop)) {
        table.append(
          row.clone()
            .append(cell.clone().text(prop))
            .append(cell.clone().text(tags[prop]))
        )
      }
    }
    exifNode.show()
  }

  function appCode() {
    return "f0fa474b804a4b6980b324bba6415875"
  }

  function updateResults (img, data) {
    var content
    if (!(img.src || img instanceof HTMLCanvasElement)) {
      content = $('<span>Loading image file failed</span>')
    } else {
      content = $('<a target="_blank">').append(img)
        .attr('download', currentFile.name)
        .attr('href', img.src || img.toDataURL())
      //todo fetch sign
      var imgurl = img.toDataURL()

      /*
      $.ajax({
        //url: "/api/sign",
        //url: "https://dm-21.data.aliyun.com/rest/160601/face/detection.json",
        url: "https://dm-24.data.aliyun.com/rest/160601/face/feature_detection.json",
        //url: "https://dm-23.data.aliyun.com/rest/160601/face/age_detection.json",
        method: "POST",
        contentType: "application/json; charset=UTF-8",
        beforeSend: function(xhr){
            xhr.setRequestHeader("Authorization", "APPCODE " + appCode())
        },
        data: JSON.stringify({
            "inputs": [
            {
                "image": {
                    "dataType": 50,
                    "dataValue": imgurl.replace(/.*;base64,(.*)/mg, "\$1") 
                }
            }
            ]
        })
      }).done(function(msg){
        var ret = JSON.parse(msg.outputs[0].outputValue.dataValue)
        var imgNode = result.find('img, canvas')
        var img = imgNode[0]
        var pixelRatio = window.devicePixelRatio || 1
        //console.log(img)
        //console.log(pixelRatio)
        imgNode.Jcrop({
        setSelect: [
          ret.rect[0] / pixelRatio,
          ret.rect[1] / pixelRatio,
          (ret.rect[0] + ret.rect[2]) / pixelRatio,
          (ret.rect[1] + ret.rect[3]) / pixelRatio
        ],
        onSelect: function (coords) {
          coordinates = coords
        },
        onRelease: function () {
          coordinates = null
        }
        })
        $.ajax({
            url: "/api/detection",
            method: "POST",
            contentType: "application/json; charset=UTF-8",
            data: JSON.stringify(ret.dense),
        }).done(function(msg){
            console.log(msg)  
            if (msg.data) {
                if( msg.data == "none") {
                    alert("系统需要和你交个朋友")
                }else {
                    alert("你好," + msg.data)
                }
            }
        })
        //todo send dense to server
      })
    }
    */
    result.children().replaceWith(content)
    if (img.getContext) {
      actionsNode.show()
    }
    if (data && data.exif) {
      displayExifData(data.exif)
    }
  }

  function displayImage (file, options) {
    currentFile = file
    if (!loadImage(
        file,
        updateResults,
        options
      )) {
      result.children().replaceWith(
        $('<span>' +
          'Your browser does not support the URL or FileReader API.' +
          '</span>')
      )
    }
  }

  function dropChangeHandler (e) {
    e.preventDefault()
    e = e.originalEvent
    var target = e.dataTransfer || e.target
    var file = target && target.files && target.files[0]
    var options = {
      maxWidth: result.width(),
      canvas: true,
      pixelRatio: window.devicePixelRatio,
      downsamplingRatio: 0.5,
      orientation: true
    }
    if (!file) {
      return
    }
    exifNode.hide()
    thumbNode.hide()
    displayImage(file, options)
  }

  // Hide URL/FileReader API requirement message in capable browsers:
  if (window.createObjectURL || window.URL || window.webkitURL ||
      window.FileReader) {
    result.children().hide()
  }

  $(document)
    .on('dragover', function (e) {
      e.preventDefault()
      e = e.originalEvent
      e.dataTransfer.dropEffect = 'copy'
    })
    .on('drop', dropChangeHandler)

  $('#file-input')
    .on('change', dropChangeHandler)

  $('#edit')
    .on('click', function (event) {
      event.preventDefault()
      var imgNode = result.find('img, canvas')
      var img = imgNode[0]
      var pixelRatio = window.devicePixelRatio || 1
      imgNode.Jcrop({
        setSelect: [
          40,
          40,
          (img.width / pixelRatio) - 40,
          (img.height / pixelRatio) - 40
        ],
        onSelect: function (coords) {
          coordinates = coords
        },
        onRelease: function () {
          coordinates = null
        }
      }).parent().on('click', function (event) {
        event.preventDefault()
      })
    })

  $('#crop')
    .on('click', function (event) {
      event.preventDefault()
      var img = result.find('img, canvas')[0]
      var pixelRatio = window.devicePixelRatio || 1
      if (img && coordinates) {
        updateResults(loadImage.scale(img, {
          left: coordinates.x * pixelRatio,
          top: coordinates.y * pixelRatio,
          sourceWidth: coordinates.w * pixelRatio,
          sourceHeight: coordinates.h * pixelRatio,
          minWidth: result.width(),
          maxWidth: result.width(),
          pixelRatio: pixelRatio,
          downsamplingRatio: 0.5
        }))
        coordinates = null
      }
    })
})
