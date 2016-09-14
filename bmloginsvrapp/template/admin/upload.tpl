{{define "Title"}}上传静态文件{{end}}
{{define "importcss"}}{{end}}
{{define "importjs"}}
<script src="/static/js/admin_upload.js"></script>
{{end}}
{{define "content"}}
<div id="id-content" class="container">
<!--Modal dialogs-->
	<div id="modalAlert" class="modal fade in" role="dialog" aria-hidden="true" style="display: none;">
		<div class="modal-dialog">
			<div class="modal-content">
				<div class="modal-header">
					<a class="close" data-dismiss="modal">×</a>
					<h3 style="color:#FE2E2E" id="id-modalalert-text">hint</h3>
				</div>
				<div class="modal-footer">
					<a href="#" class="btn" data-dismiss="modal">关闭</a>
				</div>
			</div>
		</div>
	</div>
	<div class="row">
		<div class="col-md-6 col-md-offset-3">
			{{if gt .user.Permission 3}}
			<!--Administrator panel-->
			<div style="text-align:center;">
				<form id="upload-form" enctype="multipart/form-data" action="/ajax/upload" method="post">
					<div class="form-group">
						<input id="selected-file" type="file" name="uploadfile" />
					</div>
					<label>上传类型：</label>
					<div class="form-group">
						<input type="radio" name="dst" value="static" checked>静态文件</input>
						<input type="radio" name="dst" value="tpl">页面模板</input>
					</div>
					<div class="form-group">
						<input type="input" class="form-control input-lg" placeholder="上传子路径" name="path" />
					</div>
					<a href="#" onclick="uploadFile(this)" class="btn btn-success">上传</a>
					<input id="submit-upload" type="submit" class="hide" value="上传" />
				</form>
			</div>
			{{end}}
		</div>
	</div>
</div>
{{end}}