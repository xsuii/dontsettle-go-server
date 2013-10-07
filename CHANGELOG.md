dontsettle changes
==================
## 2013-10-06
bug:
	[1] Client-side base64-encode chinese unsurport.

## 2013-9-27
bug:
	[1] Error of add history when fetch offline message by login on client side : It was too fast that couldn't add history in time. 
	[2] Can not chat with chinese, limited by base64 encode before sending. (Uncaught InvalidCharacterError: An invalid or illegal character was specified, such as in an XML name.)

## 2013-9-21
bug:
	[1]It is posible to navigate to app's another page in browser by changing url. I thought it is defect of this spa's implement(knockoutjs+sammyjs). Though it can do nothing without login and wouldn't happen on android, it looks sucks.

## 2013-9-20
bug:
	[1]localStorage use in one browser:
		I use localStorage to mark the user who is login, but while more user login, they would use the same localStorage key, though it dosen't matter at all now, but it could be better that a file identify by user's ID and store user's data.
	[2]unify client, server's time

## 2013-9-9
	* add log system(third part api : seelog)

## 2013-9-2
bug:
	[1]File transfer only surport ".txt" now and cause by the limited with download-side's JSON convert, which json.Mashal() shows "json: invalid UTF-8 in string:'xxxxx'"