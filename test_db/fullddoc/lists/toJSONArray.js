function(head, req) {
    provides('json', function() {
        var result=[];
        while (row = getRow()) {
            result.push(row.key);
        }   
        return JSON.stringify(result);
    }); 
}