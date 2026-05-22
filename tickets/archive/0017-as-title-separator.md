---
id: "0017"
status: done
type: feature
title: '# as title separator'
tags: []
created: 2026-05-22T09:02:16+02:00
updated: 2026-05-22T11:06:43.939184442+02:00
---

**fetaure to add*
add # as title separator in 'new' inline behavour

example: 
`clinban new this is title # this is body` 

this will create a new ticket with both title and body filledd by respectively. # is not parto of either, just a delimiter

the splitter should work by default, but it should be configurable - if there is `split_raw_new` set to  false in .clinban - it should not work

If the input string includes more than one splitting character , for example "Title # body with # hashes" only first segment is taken as title, rest is put to body as-is

If the input has no separator in it - just put it into body, with no title (current behavior) - if user tries to save, linter will shout.


**BUG**
but there is a problem | bug | Cobra limitation:

currently, for the present functionality where full string after 'new' should get to ticket body 

when I tried `clinban new this is title # this is body`
I got a new ticket with 'this is title' in the body. It seems Cobra truncated the input on #

Notice - I did not put the string in quotes, it was just a stream of words following `new`


