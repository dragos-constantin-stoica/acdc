function user_context(userctx, secobj) {
  var is_admin = function() {
    return userctx.indexOf('_admin') != -1;
  }
  return {'is_admin': is_admin}
}

exports['user'] = user_context