// Time-stamp: <2022-10-24 18:58:09 krylon>
// -*- mode: javascript; coding: utf-8; -*-
// Copyright 2015-2020 Benjamin Walkenhorst <krylon@gmx.net>
//
// This file has grown quite a bit larger than I had anticipated.
// It is not a /big/ problem right now, but in the long run, I will have to
// break this thing up into several smaller files.

'use strict'

function defined (x) {
    return undefined !== x && null !== x
}

function fmtDateNumber (n) {
    return (n < 10 ? '0' : '') + n.toString()
} // function fmtDateNumber(n)

function timeStampString (t) {
    if ((typeof t) === 'string') {
        return t
    }

    const year = t.getYear() + 1900
    const month = fmtDateNumber(t.getMonth() + 1)
    const day = fmtDateNumber(t.getDate())
    const hour = fmtDateNumber(t.getHours())
    const minute = fmtDateNumber(t.getMinutes())
    const second = fmtDateNumber(t.getSeconds())

    const s =
          year + '-' + month + '-' + day +
          ' ' + hour + ':' + minute + ':' + second
    return s
} // function timeStampString(t)

function fmtDuration (seconds) {
    let minutes = 0
    let hours = 0

    while (seconds > 3599) {
        hours++
        seconds -= 3600
    }

    while (seconds > 59) {
        minutes++
        seconds -= 60
    }

    if (hours > 0) {
        return `${hours}h${minutes}m${seconds}s`
    } else if (minutes > 0) {
        return `${minutes}m${seconds}s`
    } else {
        return `${seconds}s`
    }
} // function fmtDuration(seconds)

function beaconLoop () {
    try {
        if (settings.beacon.active) {
            const req = $.get('/ajax/beacon',
                              {},
                              function (response) {
                                  let status = ''

                                  if (response.Status) {
                                      status = 
                                          response.Message +
                                          ' running on ' +
                                          response.Hostname +
                                          ' is alive at ' +
                                          response.Timestamp
                                  } else {
                                      status = 'Server is not responding'
                                  }

                                  const beaconDiv = $('#beacon')[0]

                                  if (defined(beaconDiv)) {
                                      beaconDiv.innerHTML = status
                                      beaconDiv.classList.remove('error')
                                  } else {
                                      console.log('Beacon field was not found')
                                  }
                              },
                              'json'
                             ).fail(function () {
                                 const beaconDiv = $('#beacon')[0]
                                 beaconDiv.innerHTML = 'Server is not responding'
                                 beaconDiv.classList.add('error')
                                 // logMsg("ERROR", "Server is not responding");
                             })
        }
    } finally {
        window.setTimeout(beaconLoop, settings.beacon.interval)
    }
} // function beaconLoop()

function beaconToggle () {
    settings.beacon.active = !settings.beacon.active
    saveSetting('beacon', 'active', settings.beacon.active)

    if (!settings.beacon.active) {
        const beaconDiv = $('#beacon')[0]
        beaconDiv.innerHTML = 'Beacon is suspended'
        beaconDiv.classList.remove('error')
    }
} // function beaconToggle()

/*
  The ‘content’ attribute of Window objects is deprecated.  Please use ‘window.top’ instead. interact.js:125:8
  Ignoring get or set of property that has [LenientThis] because the “this” object is incorrect. interact.js:125:8

*/

function db_maintenance () {
    const maintURL = '/ajax/db_maint'

    const req = $.get(
        maintURL,
        {},
        function (res) {
            if (!res.Status) {
                console.log(res.Message)
                postMessage(new Date(), 'ERROR', res.Message)
            } else {
                const msg = 'Database Maintenance performed without errors'
                console.log(msg)
                postMessage(new Date(), 'INFO', msg)
            }
        },
        'json'
    ).fail(function () {
        const msg = 'Error performing DB maintenance'
        console.log(msg)
        postMessage(new Date(), 'ERROR', msg)
    })
} // function db_maintenance()

function msgCheckSum (timestamp, level, msg) {
    const line = [timeStampString(timestamp), level, msg].join('##')

    const cksum = sha512(line)
    return cksum
}

let curMessageCnt = 0

function post_test_msg () {
    const user = $('#msgTestText')[0]
    const msg = user.value
    const now = new Date()

    postMessage(now, 'DEBUG', msg)
} // function post_tst_msg()

function postMessage (timestamp, level, msg) {
    const row = '<tr id="msg_' +
          msgCheckSum(timestamp, level, msg) +
          '"><td>' +
          timeStampString(timestamp) +
          '</td><td>' +
          level +
          '</td><td>' +
          msg +
          '</td></tr>\n'

    msgRowAdd(row)
} // function postMessage(timestamp, level, msg)

function adjustMsgMaxCnt () {
    const cntField = $('#max_msg_cnt')[0]
    const newMax = cntField.valueAsNumber

    if (newMax < curMessageCnt) {
        const rows = $('#msg_body')[0].children

        while (rows.length > newMax) {
            rows[rows.length - 1].remove()
            curMessageCnt--
        }
    }

    saveSetting('messages', 'maxShow', newMax)
} // function adjustMaxMsgCnt()

function adjustMsgCheckInterval () {
    const intervalField = $('#msg_check_interval')[0]
    if (intervalField.checkValidity()) {
        const interval = intervalField.valueAsNumber
        // intervalField.setInterval(interval); // ???
        saveSetting('messages', 'interval', interval)
    }
} // function adjustMsgCheckInterval()

function toggleCheckMessages () {
    const box = $('#msg_check_switch')[0]
    const newVal = box.checked

    saveSetting('messages', 'queryEnabled', newVal)
} // function toggleCheckMessages()

function getNewMessages () {
    // const msgURL = '/ajax/get_messages'

    // try {
    //     if (!settings.messages.queryEnabled) {
    //         return
    //     }

    //     const req = $.get(
    //         msgURL,
    //         {},
    //         function (res) {
    //             if (!res.Status) {
    //                 const msg = msgURL +
    //                       ' failed: ' +
    //                       res.Message

    //                 console.log(msg)
    //                 alert(msg)
    //             } else {
    //                 let i = 0
    //                 for (i = 0; i < res.Messages.length; i++) {
    //                     const item = res.Messages[i]
    //                     const rowid =
    //                           'msg_' +
    //                           msgCheckSum(item.Time, item.Level, item.Message)
    //                     const row = '<tr id="' +
    //                           rowid +
    //                           '"><td>' +
    //                           item.Time +
    //                           '</td><td>' +
    //                           item.Level +
    //                           '</td><td>' +
    //                           item.Message +
    //                           '</td><td>' +
    //                           '<input type="button" value="Delete" onclick="msgRowDelete(\'' +
    //                           rowid +
    //                           '\');" />' +
    //                           '</td></tr>\n'

    //                     msgRowAdd(row)
    //                 }
    //             }
    //         },
    //         'json'
    //     )
    // } finally {
    //     window.setTimeout(getNewMessages, settings.messages.interval)
    // }
} // function getNewMessages()

function logMsg (level, msg) {
    const timestamp = timeStampString(new Date())
    const rowID = 'msg_' + sha512(msgCheckSum(timestamp, level, msg))
    const row = '<tr id="' +
          rowID +
          '"><td>' +
          timestamp +
          '</td><td>' +
          level +
          '</td><td>' +
          msg +
          '</td><td>' +
          '<input type="button" value="Delete" onclick="msgRowDelete(\'' +
          rowID +
          '\');" />' +
          '</td></tr>\n'

    $('#msg_display_tbl')[0].innerHTML += row
} // function logMsg(level, msg)

function msgRowAdd (row) {
    const msgBody = $('#msg_body')[0]

    msgBody.innerHTML = row + msgBody.innerHTML

    if (++curMessageCnt > settings.messages.maxShow) {
        msgBody.children[msgBody.children.length - 1].remove()
    }

    const tbl = $('#msg_tbl')[0]
    if (tbl.hidden) {
        tbl.hidden = false
    }
} // function msgRowAdd(row)

function msgRowDelete (rowID) {
    const row = $('#' + rowID)[0]

    if (row != undefined) {
        row.remove()
        if (--curMessageCnt == 0) {
            const tbl = $('#msg_tbl')[0]
            tbl.hidden = true
        }
    }
} // function msgRowDelete(rowID)

function msgRowDeleteAll () {
    const msgBody = $('#msg_body')[0]
    msgBody.innerHTML = ''
    curMessageCnt = 0

    const tbl = $('#msg_tbl')[0]
    tbl.hidden = true
} // function msgRowDeleteAll()

function requestTestMessages () {
    const urlRoot = '/ajax/rnd_message/'

    const cnt = $('#msg_cnt')[0].valueAsNumber
    const rounds = $('#msg_round_cnt')[0].valueAsNumber
    const delay = $('#msg_round_delay')[0].valueAsNumber

    if (cnt == 0) {
        console.log('Generate *0* messages? Alrighty then...')
        return
    }

    const reqURL = urlRoot + cnt

    $.get(
        reqURL,
        {
            Rounds: rounds,
            Delay: delay
        },
        (res) => {
            if (!res.Status) {
                console.log(res.Message)
                alert(res.Message)
            }
        },
        'json'
    ).fail(function () {
        const msg = 'Requesting test messages failed.'
        console.log(msg)
        // alert(msg);
        logMsg('ERROR', msg)
    })
} // function requestTestMessages()

function toggleMsgTestDisplayVisible () {
    const tbl = $('#test_msg_cfg')[0]

    if (tbl.hidden) {
        tbl.hidden = false

        const checkbox = $('#msg_check_switch')[0]
        settings.messages.queryEnabled = checkbox.checked
    } else {
        settings.messages.queryEnabled = false
        tbl.hidden = true
    }
} // function toggleMsgTmpDisplayVisible()

function toggleMsgDisplayVisible () {
    const display = $('#msg_display_div')[0]

    display.hidden = !display.hidden
} // function toggleMsgDisplayVisible()

function rate_item (item_id, new_rating) {
    const req = $.post('/ajax/rate_item',
                       { ID: item_id, Rating: new_rating },
                       function (reply) {
                           if (!reply.Status) {
                               const msg = `Error rating Item #${item_id}: ${reply.Message}`
                               console.log(msg)
                               alert(msg)
                               return
                           }
                           let content = ''
                           const row_id = `#item_${item_id}`
                           const row = $(row_id)
                           if (new_rating <= 0.0) {
                               content = '<img src="/static/emo_boring.png" />'
                               row.addClass('boring')
                           } else {
                               content = '<img src="/static/emo_interesting.png" />'
                               row.removeClass('boring')
                           }

                           content += `<br /><input
        type="button"
        class="btn btn-secondary"
        value="Unvote"
        onclick="unvote_item(${item_id});" />`

                           $('#item_rating_' + item_id)[0].innerHTML = content
                       },
                       'json')

    req.fail(function (reply, status_text, xhr) {
        console.log('Our Ajax request failed: ' + status_text)
        const data = reply // $.parseJSON(reply.responseText);
        if (data.Status) {
            var msg = 'Error rating item - but Status is true?!?!?!'
            alert(msg)
            console.log(msg)
        } else {
            var msg = 'Error rating item - ' + data.Message
            alert(msg)
            console.log(msg)
        }
    })
} // function rate_item(item_id, new_rating)

function unvote_item (item_id) {
    const addr = '/ajax/unrate_item/' + item_id
    const req = $.get(
        addr,
        {},
        function (reply) {
            if (!reply.Status) {
                const msg = `Error retracting vote for item ${item_id}: ${reply.Message}`
                console.log(msg)
                alert(msg)
                return
            }

            // It would be nice if I could display the suggested Rating, too!
            const row_id = `#item_${item_id}`
            const cell_id = `#item_rating_${item_id}`

            const content = `<input
        class="btn btn-secondary"
        type="button"
        value="Interesting"
        onclick="rate_item(${item_id}, 1);" />
        <br />&nbsp;<br />
        <input
        type="button"
        class="btn btn-secondary"
        value="Booooring"
        onclick="rate_item(${item_id}, 0);" />
        <br />
`

            const cell = $(cell_id)[0]
            cell.innerHTML = content

            $(row_id).removeClass('boring')
            console.log('Rating on Item ' + item_id + ' has been cleared.')

            const r = $.get(
                `/ajax/suggest_rating/${item_id}`,
                {},
                (reply) => {
                    if (!reply.Status) {
                        const msg = `Error requesting Rating for Item ${item_id}: ${reply.Message}`
                        console.log(msg)
                        alert(msg)
                        return
                    }

                    const filename = reply.Rating < 0 ? 'emo_boring.png' : 'emo_interesting.png'
                    const img = `<small>${reply.Rating}<small><br /><img src="/static/${filename}" />`

                    cell.innerHTML += img
                },
                'json')

            r.fail(function (rep, stat, x) {
                const msg = 'Error unrating Item at ' + addr + ': ' + status_text
                console.log(msg)
                alert(msg)
            })
        },
        'json')

    req.fail(function (reply, status_text, xhr) {
        const msg = 'Error unrating Item at ' + addr + ': ' + status_text
        console.log(msg)
        alert(msg)
    })
} // function unvote_item(item_id)

function hide_boring_items () {
    console.log('Hiding boring items.')
    $.each($('tr.boring'), () => { $(this).hide() })
} // function hide_boring_items()

function show_boring_items () {
    console.log('Displaying boring items.')
    $.each($('tr.boring'), function () { $(this).show() })
} // function show_boring_items()

function toggle_hide_boring () {
    console.log('toggle_hide_boring()')

    settings.items.hideboring = !settings.items.hideboring
    saveSetting('items', 'hideboring', settings.items.hideboring)

    if (settings.items.hideboring) {
        hide_boring_items()
    } else {
        show_boring_items()
    }

    return true
} // function toggle_hide_boring()

function rebuildFTS () {
    const req = $.get('/ajax/rebuild_fts',
                      '',
                      function (reply) {
                          console.log('FTS index has been rebuilt.')
                      })

    req.fail(function (reply, status_text, xhr) {
        const msg = reply + ' -- ' + status_text
        console.log(msg)
        alert(msg)
    })
} // function rebuildFTS()

function attach_tag (form_id, item_id) {
    const id = `#${form_id}`
    const sel = $(id)[0].value
    const tag_id = parseInt(sel)

    console.log('Attach Tag #' + sel + ' to Item ' + item_id + '.')

    const req = $.post('/ajax/tag_link_create',
                       { Tag: tag_id, Item: item_id },
                       function (reply) {
                           console.log(`Successfully attached Tag ${sel} to Item ${item_id}`)
                           const div_id = `#tags_${item_id}`
                           const div = $(div_id)[0]

                           const tag = `<a class="item_${item_id}_tag_${tag_id}" href="/tag/${tag_id}">${reply.Name}</a>&nbsp;<img class="item_${item_id}_tag_${tag_id}" src="/static/delete.png" role="button" onclick="untag(${item_id}, ${tag_id});" /> &nbsp; `

                           div.innerHTML += tag

                           const opt_id = `#${form_id}_opt_${tag_id}`
                           //$(opt_id).hide()
                           $(opt_id)[0].disabled = true
                           const form = $(id)[0]
                           for (const [idx, val] of Object.entries(form.options)) {
                               if (val.style.display != 'none') {
                                   form.value = val.value
                                   break
                               }
                           }
                       },
                       'json')

    req.fail(function (reply, status_text, xhr) {
        console.log(`Error attaching Tag to Item: ${status_text} // ${reply}`)
    })
} // function attach_tag(form_id, item_id)

function quick_tag (item_id, tag_id, button_id) {
    const req = $.post('/ajax/tag_link_create',
                       { Tag: tag_id, Item: item_id },
                       (reply) => {
                           if (!reply.Status) {
                               const msg = `Error tagging Item ${item_id}: ${reply.Message}`
                               console.log(msg)
                               alert(msg)
                               return
                           }

                           console.log(`Successfully attached Tag ${tag_id} to Item ${item_id}`)
                           const div_id = `#tags_${item_id}`
                           const div = $(div_id)[0]
                           const form_id = `tag_menu_item_${item_id}`

                           const tag = `<a class="item_${item_id}_tag_${tag_id}" href="/tag/${tag_id}">${reply.Name}</a>&nbsp;<img class="item_${item_id}_tag_${tag_id}" src="/static/delete.png" role="button" onclick="untag(${item_id}, ${tag_id});" /> &nbsp; `

                           div.innerHTML += tag

                           const opt_id = `#${form_id}_opt_${tag_id}`
                           // $(opt_id).hide()
                           $(opt_id)[0].disabled = true

                           $(button_id)[0].onclick = null
                       },
                       'json')

    req.fail(function (reply, status_text, xhr) {
        const msg = `Error attaching Tag to Item: ${status_text} // ${reply}`
        console.log(msg)
        alert(msg)
    })
} // function quick_tag (item_id, tag_id)

function untag (item_id, tag_id) {
    const tag = `#item_${item_id}_tag_${tag_id}`
    const msg = `Remove tag ${tag_id} from Item ${item_id}`
    console.log(msg)

    const req = $.post('/ajax/tag_link_delete',
                       { Tag: tag_id, Item: item_id },
                       function (reply) {
                           console.log(`Successfully detached Tag ${tag_id} from Item ${item_id}`)

                           const label_id = `.item_${item_id}_tag_${tag_id}`
                           const labels = $(label_id)

                           labels.each(function () { $(this).remove() })

                           const sel_id = `tag_menu_item_${item_id}`
                           const opt_id = `#${sel_id}_opt_${tag_id}`
                           // $(opt_id).show()
                           $(opt_id)[0].disabled = false
                       },
                       'json')

    req.fail(function (reply, status_text, xhr) {
        const errmsg = `Error attaching Tag to Item: ${status_text} // ${reply}`
        console.log(errmsg)
        alert(errmsg)
    })
} // function untag(item_id, tag_id)

// "/ajax/read_later_mark"

function read_later_show (item_id) {
    const button_id = `#read_later_button_${item_id}`
    const form_id = `#read_later_form_${item_id}`

    $(form_id).show()
    $(button_id).hide()
} // function read_later_show(item_id)

function read_later_reset (item_id) {
    const button_id = `#read_later_button_${item_id}`
    const form_id = `#read_later_form_${item_id}`

    $(form_id).hide()
    $(button_id).show()
} // function read_later_reset(item_id)

function read_later_mark (item_id) {
    console.log(`IMPLEMENTME: Mark Item ${item_id} for later reading.`)
    const button_id = `#read_later_button_${item_id}`
    const form_id = `#read_later_form_${item_id}`
    const num_id = `#later_deadline_num_${item_id}`
    const unit_sel_id = `#later_deadline_unit_${item_id}`
    const note_id = `#later_note_${item_id}`

    const num = $(num_id)[0].value
    const unit = $(unit_sel_id)[0].value

    const deadline = num * unit

    const now = new Date()
    const due_time = Math.floor(now.getTime() / 1000 + deadline)

    const note = $(note_id)[0].value

    const req = $.post('/ajax/read_later_mark',
                       {
                           ItemID: item_id,
                           Note: note,
                           Deadline: due_time
                       },
                       function (reply) {
                           if (!reply.Status) {
                               const errmsg = `Error marking Item for later: ${reply.Message}`
                               console.log(errmsg)
                               alert(errmsg)
                           } else {
                               $(form_id).hide()
                               $(button_id).hide()
                           }
                       },
                       'json')

    req.fail(function (reply, status_text, xhr) {
        console.log(`Error attaching Tag to Item: ${status_text} // ${reply}`)
    })

    console.log(`Deadline is ${due_time}`)

    $(form_id).hide()
    $(button_id).show()
} // function read_later_mark(item_id)

function read_later_mark_read (item_id, item_title) {
    const checkbox_id = `#later_mark_read_${item_id}`
    const state = $(checkbox_id)[0].checked
    const url = `/ajax/read_later_set_read/${item_id}/${state ? 1 : 0}`

    const req = $.get(url,
                      {},
                      function (reply) {
                          if (!reply.Status) {
                              const errmsg = `Error marking Item as read: ${reply.Message}`
                              console.log(errmsg)
                              alert(errmsg)
                          } else {
                              // Do something!
                              const rowid = `#item_${item_id}`
                              if (state) {
                                  $(rowid).addClass('read')
                                  $(rowid).removeClass('urgent')
                              } else {
                                  // Instead of just turning on the urgent class,
                                  // we should if the item's deadline has actually
                                  // passed. What would be the easiest way of
                                  // doing that?
                                  const now = new Date()
                                  const row_id = `#item_${item_id}`
                                  const cell = $(row_id)[0].children[0]
                                  const txt = cell.textContent.trim()
                                  const deadline = new Date(txt)

                                  $(rowid).removeClass('read')
                                  if (deadline <= now) {
                                      $(rowid).addClass('urgent')
                                  }
                              }
                          }
                      },
                      'json')

    req.fail(function (reply, status_text, xhr) {
        console.log(`Error marking Item as Read: ${status_text} // ${reply}`)
        $(checkbox_id)[0].checked = !state
    })
} // function read_later_mark_read(item_id, item_title)

function read_later_toggle_read_entries () {
    const checkbox_id = '#hide_old'
    const state = $(checkbox_id)[0].checked
    const query = '#read_later_list .read'

    if (state) {
        $(query).hide()
    } else {
        $(query).show()
    }
} // function read_later_toggle_read_entries()

function edit_feed (feed_id) {
    console.log(`IMPLEMENTME: edit_feed(${feed_id})`)
    const form_id = '#feed_form'
    const feed = feeds[feed_id]

    $('#form_url')[0].value = feed.url
    $('#form_name')[0].value = feed.name
    $('#form_homepage')[0].value = feed.homepage
    $('#form_interval')[0].value = feed.interval / 60
    // $("#form_active")[0].checked = feed.active;
    $('#form_id')[0].value = feed.id
    $(form_id).show()
} // function edit_feed(feed_id)

function feed_form_submit () {
    console.log('IMPLEMENTME: feed_form_submit()')

    const id = $('#form_id')[0].value
    const name = $('#form_name')[0].value
    const url = $('#form_url')[0].value
    const homepage = $('#form_homepage')[0].value
    const interval = $('#form_interval')[0].value
    // const active = $("#form_active")[0].checked;

    const feed = feeds[id]

    const req = $.post('/ajax/feed_update',
                       {
                           ID: id,
                           Name: name,
                           URL: url,
                           Homepage: homepage,
                           Interval: interval * 60
                           // "Active": active,
                       },
                       function (reply) {
                           if (reply.Status) {
                               console.log(`Successfully updated Feed ${name}`)

                               const hp = $(`#homepage_${id}`)[0]
                               hp.href = homepage
                               hp.innerHTML = name

                               const lnk = $(`#url_${id}`)[0]
                               lnk.href = url
                               lnk.innerHTML = url

                               $(`#interval_${id}`)[0].innerHTML = fmtDuration(interval * 60)

                               $('#feed_form').hide()
                           } else {
                               const msg = `Error updating Feed ${name}: ${reply.Message}`
                               console.log(msg)
                               alert(msg)
                           }
                       },
                       'json')

    req.fail(function (reply, status_text, xhr) {
        console.log(`Error updating Feed: ${status_text} // ${reply}`)
        $(checkbox_id)[0].checked = !state
    })
} // function feed_submit()

function feed_form_reset () {
    $('#feed_form')[0].reset()
    $('#feed_form').hide()
} // function feed_form_reset()

function toggle_feed_active (feed_id) {
    const checkbox_id = `#feed_active_${feed_id}`
    const active = $(checkbox_id)[0].checked

    const req = $.get(`/ajax/feed_set_active/${feed_id}/${active}`,
                      {},
                      function (reply) {
                          if (!reply.Status) {
                              $(checkbox_id)[0].checked = !active
                              alert(reply.Message)
                          }
                      },
                      'json')

    req.fail(function (reply, status_text, xhr) {
        console.log(`Error toggling Feed ${feed_id}: ${status_text} // ${reply}`)
        $(checkbox_id)[0].checked = !active
    })
} // function toggle_feed_active(feed_id)

function display_tag_items (tag_id) {
    const url = `/ajax/items_by_tag/${tag_id}`

    const req1 = $.post(url,
                        {},
                        function (reply) {
                            if (reply.Status) {
                                $('#item_div')[0].innerHTML = reply.Message
                                shrink_images()
                            } else {
                                console.log(reply.Message)
                                alert(reply.Message)
                            }
                        },
                        'json')

    req1.fail(function (reply, status_text, xhr) {
        console.log(`Error getting Items: ${status_text} - ${xhr}`)
    })

    const req2 = $.get(`/ajax/tag_details/${tag_id}`,
                       {},
                       (reply) => {
                           if (reply.Status) {
                               $('#tag_id')[0].value = reply.Tag.ID
                               $('#tag_name')[0].value = reply.Tag.Name
                               $('#tag_description')[0].value = reply.Tag.Description
                               console.log(`Server said ${reply.Message}`)

                               const sel = $('#tag_parent')[0]
                               for (const [idx, entry] of Object.entries(sel.options)) {
                                   if (entry.value == reply.Tag.Parent) {
                                       sel.value = entry.value
                                       break
                                   }
                               }
                           }
                       },
                       'json')

    req2.fail((reply, status_text, xhr) => { console.log(`Error getting Tag: ${status_text} - ${xhr}`) })
} // function display_tag_items(tag_id)

// Found here: https://stackoverflow.com/questions/3971841/how-to-resize-images-proportionally-keeping-the-aspect-ratio#14731922
function shrink_img (srcWidth, srcHeight, maxWidth, maxHeight) {
    const ratio = Math.min(maxWidth / srcWidth, maxHeight / srcHeight)

    return { width: srcWidth * ratio, height: srcHeight * ratio }
} // function shrink_img(srcWidth, srcHeight, maxWidth, maxHeight)

function shrink_images () {
    const selector = 'table.items img'
    const maxHeight = 300
    const maxWidth = 300

    $(selector).each(function () {
        const img = $(this)[0]
        if (img.width > maxWidth || img.height > maxHeight) {
            const size = shrink_img(img.width, img.height, maxWidth, maxHeight)

            img.width = size.width
            img.height = size.height
        }
    })
} // function shrink_images()

function load_feed_items (feed_id) {
    const div_id = '#item_div'
    const url = `/ajax/items_by_feed/${feed_id}`

    const req = $.get(url,
                      {},
                      function (reply) {
                          if (reply.Status) {
                              $('#item_div')[0].innerHTML = reply.Message
                              shrink_images()
                          } else {
                              console.log(reply.Message)
                              alert(reply.Message)
                          }
                      },
                      'json')

    req.fail(function (reply, status_text, xhr) {
        console.log(`Error getting Items: ${status_text} - ${xhr}`)
    })
} // function load_feed_items(feed_id)

function shutdown_server () {
    const url = '/ajax/shutdown'

    if (!confirm('Shut down server?')) {
        return false
    }

    const req = $.get(url,
                      { AreYouSure: true, AreYouReallySure: true },
                      function (reply) {
                          if (!reply.Status) {
                              const msg = `Error shutting down Server: ${reply.Message}`
                              console.log(msg)
                              alert(msg)
                          }
                      },
                      'json')

    req.fail(function (reply, status_text, xhr) {
        const msg = `Error getting Items: ${status_text} - ${xhr}`
        console.log(msg)
        alert(msg)
    })
} // function shutdown_server()

function items_go_page () {
    const idx = $('#choose_page')[0].value
    const addr = `/items/${idx}`

    window.location = addr
} // function items_go_page()

function item_add_cluster (item_id, clu) {
    const addr = '/ajax/cluster_link_add'

    let req = $.post(
        addr,
        { ItemID: item_id, ClusterID: clu.ID },
        (reply) => {
            console.log(reply)
            if (reply.Status) {
                const div_id = `#cluster_list_${item_id}`
                const input_id = `#cluster_input_${item_id}`
                const elt = `<span id="cluster_link_${item_id}_${clu.ID}">
  <a href="/cluster/${clu.ID}">${clu.Name}</a>
  (${clu.ID})
  <img id="item_${item_id}_${clu.ID}"
       src="/static/delete.png"
       role="button"
       onclick="item_rm_cluster(${item_id}, ${clu.ID});"
       />
</span>
<br />
`
                $(div_id)[0].innerHTML += elt
                $(input_id)[0].value = ''
            } else {
                const msg = `Error adding Item ${item_id} to Cluster ${clu.ID}: ${reply.Message}`
                console.log(msg)
                alert(msg)
            }
        },
        'json'
    )

    req.fail = function(rep, stat, xhr) {
        const msg = `Error linking Item ${item_id} to Cluster ${cluster_id}: ${rep} / ${stat} / ${xhr}`;
        console.log(msg);
        alert(msg);
    };
} // function item_add_cluster (div_id, clu)

function make_cluster_key_handler (id) {
    return (e) => {
        const itemID = id;
        const inputID = `#cluster_input_${id}`;
        let status = false;
        if (e.code == 'Enter') {
            const name = $(inputID)[0].value;
            let clu = clusterList[name];
            console.log(`Add Item ${id} to Cluster ${clu ? clu.Name + "!" : name}`);
            if (clu == undefined) {
                // create Cluster!!!
                const desc = '' // prompt(`Description for ${name}?`);

                const req = $.post(
                    "/ajax/cluster_create",
                    { Name: name, Description: desc },
                    (reply) => {
                        if (reply.Status) {
                            clusterList[name] = reply.Cluster
                            status = true
                            clu = reply.Cluster
                            item_add_cluster(itemID, clu)
                            const list_item = `<option value="${name}"></option>`
                            const list = $('#cluster_list')[0]
                            list.innerHTML += list_item
                        } else {
                            const msg = `Error creating cluster "${name}": ${reply.Message}`;
                            console.log(msg);
                            alert(msg);
                        }
                    },
                    "json"
                )

                req.fail = (rep, stat, xhr) => {
                    console.log(`Error creating Cluster ${name}: ${rep}: ${stat} | ${xhr}`);
                }
            } else {
                console.assert(clu != undefined && clu != null)
                item_add_cluster(itemID, clu)
            }
        }
    }
} // function make_cluster_key_handler (itemID)

function install_cluster_key_handler (id) {
    const inpID = `#cluster_input_${id}`
    const elt = $(inpID)[0]

    elt.onkeydown = make_cluster_key_handler(id)
} // function install_cluster_key_handler (id)

function item_rm_cluster (item_id, cluster_id) {
    const addr = "/ajax/cluster_link_del"
    const req = $.post(
        addr,
        { ItemID: item_id, ClusterID: cluster_id },
        (reply) => {
            if (reply.Status) {
                const eltID = `#cluster_link_${item_id}_${cluster_id}`
                let elt = $(eltID)[0]
                elt.remove()
            } else {
                const msg = `Error unlinking Item ${item_id} from Cluster ${cluster_id}: ${reply.Message}`
                console.error(msg)
                alert(msg)
            }
        },
        "json")

    req.fail = (rep, stat, xhr) => {
        console.error(`Error creating Cluster ${name}: ${rep}: ${stat} | ${xhr}`)
    }
} // function item_rm_cluster (item_id, cluster_id)

function cluster_display_items (cluster_id) {
    const url = '/ajax/cluster_items'
    const div = '#cluster_item_div'
    // alert('cluster_display_items: IMPLEMENT ME!!!')

    const req = $.post(
        url,
        { ClusterID: cluster_id },
        (reply) => {
            if (reply.Status) {
                $(div)[0].innerHTML = reply.HTML
            } else {
                const msg = `Failed to load Items for Cluster #${cluster_id}: ${reply.Message}`
                console.error(msg)
                alert(msg)
            }
        },
        "json"
    )

    req.fail = (rep, stat, xhr) => {
        console.error(`Error fetching Items for Cluster ${cluster_id}: ${rep}: ${stat} | ${xhr}`)
    }
} // function cluster_display_items (cluster_id)

function download_item (item_id) {
    const url = '/ajax/download_item'
    const div_id = `#download_item_${item_id}`
    const div = $(div_id)[0]

    const req = $.post(
        url,
        { ItemID: item_id },
        (reply) => {
            if (reply.Status) {
                // Strictly speaking, at this point we just *requested* the
                // server to download the Item, it might still fail.
                // I'm thinking of using websockets to notify the frontend
                // when the page has been downloaded successfully or when
                // something goes wrong.
                const content = `<a href="/archive/${item_id}/index.html">Archive</a>`
                div.innerHTML = content
            } else {
                const msg = `Error requesting download of Item ${item_id}: ${reply.Message}`
                console.error(msg)
                alert(msg)
            }
        },
        'json'
    )

    req.fail = (rep, stat, xhr) => {
        console.error(`Error requesting download of ${item_id}: ${rep} / ${stat} / ${xhr}`)
    }
} // function download_item (item_id)

let dl_item_id = 0

function load_archived_page (page_id) {
    const frameID = "#page_frame"
    const frame   = $(frameID)[0]
    const addr    = `/archive/${page_id}/index.html`
    if (page_id != dl_item_id) {
        dl_item_id = page_id
        frame.src = addr
    } else {
        frame.src = 'about:blank'
    }
} // function load_archived_page(page_id)

function archive_delete (item_id) {
    const url = `/ajax/archive_delete/${item_id}`

    const req = $.post(url,
                       {},
                       (reply) => {
                           if (reply.Status) {
                               const list_id = `#item_${item_id}`
                               $(list_id).remove()
                               if (dl_item_id == item_id) {
                                   dl_item_id = 0
                                   $('#page_frame')[0].src = 'about:blank'
                               }
                           } else {
                               const msg = `Error deleting downloaded Item ${item_id}: ${reply.Message}`
                               console.error(msg)
                               alert(msg)
                           }
                       },
                       'json')

    req.fail = (rep, stat, xhr) => {
        console.error(`Error requesting deletion of downloaded Item ${item_id}: ${rep} / ${stat} / ${xhr}`)
    }
} // function archive_delete(item_id)

function page_frame_resize () {
    const h = window.innerHeight
    const w = window.innerWidth
    const frameID = "#page_frame"
    const frame = $(frameID)[0]

    const msg = `Window size is ${w}x${h}`

    console.log(msg)
    alert(msg)
} // function page_frame_resize ()
