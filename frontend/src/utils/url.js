function removeLastDir (url) {
  var arr = url.split('/')
  if (arr.pop() === '') {
    arr.pop()
  }

  return arr.join('/')
}

// this code borrow from mozilla
// https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/encodeURIComponent#Examples
function encodeRFC5987ValueChars(str) {
  return encodeURIComponent(str).
      // Note that although RFC3986 reserves "!", RFC5987 does not,
      // so we do not need to escape it
      replace(/['()]/g, escape). // i.e., %27 %28 %29
      replace(/\*/g, '%2A').
          // The following are not required for percent-encoding per RFC5987, 
          // so we can allow for a little better readability over the wire: |`^
          replace(/%(?:7C|60|5E)/g, unescape);
}

function encodePath(str) {
  return str.split('/').map(v => encodeURIComponent(v)).join('/')
}

function unicodeToChar(text) {
  return text.replace(/\\u[\dA-F]{4}/gi, 
         function (match) {
              return String.fromCharCode(parseInt(match.replace(/\\u/g, ''), 16));
         });
}

export default {
  encodeRFC5987ValueChars: encodeRFC5987ValueChars,
  removeLastDir: removeLastDir,
  encodePath: encodePath,
  unicodeToChar: unicodeToChar
}
