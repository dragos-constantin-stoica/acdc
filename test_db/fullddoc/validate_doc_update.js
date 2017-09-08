function(newDoc, oldDoc, userCtx) {
  if (newDoc.type === undefined) {
      throw({forbidden: 'Document must have a type.'});
  }
  if (newDoc.author) {
    enforce(newDoc.author == userCtx.name, 'You may only update documents with author ' + userCtx.name);
  }
  //see views lib export for CommonJS module
  user = require('lib/security').user(userctx, secobj);
  return user.is_admin();
}