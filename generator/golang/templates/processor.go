// Copyright 2021 CloudWeGo Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package templates

// Processor .
var Processor = `
{{define "Processor"}}
{{- UseStdLibrary "thrift" "context"}}
{{- $BasePrefix := ServicePrefix .Base}}
{{- $BaseService := ServiceName .Base}}
{{- $ServiceName := .GoName}}
{{- $ProcessorName := printf "%s%s" $ServiceName "Processor"}}
{{- if .Extends}}
type {{$ProcessorName}} struct {
	*{{$BasePrefix}}{{$BaseService}}Processor
}
{{- else}}
type {{$ProcessorName}} struct {
	processorMap map[string]thrift.TProcessorFunction
	handler      {{$ServiceName}}
}

func (p *{{$ProcessorName}}) AddToProcessorMap(key string, processor thrift.TProcessorFunction) {
	p.processorMap[key] = processor
}

func (p *{{$ProcessorName}}) GetProcessorFunction(key string) (processor thrift.TProcessorFunction, ok bool) {
	processor, ok = p.processorMap[key]
	return processor, ok
}

func (p *{{$ProcessorName}}) ProcessorMap() map[string]thrift.TProcessorFunction {
	return p.processorMap
}
{{- end}}

func New{{$ProcessorName}}(handler {{$ServiceName}}) *{{$ProcessorName}} {
	{{- if .Extends}}
	self := &{{$ProcessorName}}{ {{$BasePrefix}}New{{$BaseService}}Processor(handler) }
	{{- else}}
	self := &{{$ProcessorName}}{handler: handler, processorMap: make(map[string]thrift.TProcessorFunction)}
	{{- end}}
	{{- range .Functions}}
	self.AddToProcessorMap("{{.Name}}", &{{$ProcessorName | Unexport}}{{.GoName}}{handler: handler})
	{{- end}}
	return self
}

{{- if not .Extends}}
func (p *{{$ProcessorName}}) Process(ctx context.Context, iprot, oprot thrift.TProtocol) (success bool, err thrift.TException) {
	name, _, seqId, err2 := iprot.ReadMessageBegin(ctx)
	if err2 != nil {
		return false, thrift.WrapTException(err2)
	}
	if processor, ok := p.GetProcessorFunction(name); ok {
		return processor.Process(ctx, seqId, iprot, oprot)
	}
	iprot.Skip(ctx, thrift.STRUCT)
	iprot.ReadMessageEnd(ctx)
	x := thrift.NewTApplicationException(thrift.UNKNOWN_METHOD, "Unknown function "+name)
	oprot.WriteMessageBegin(ctx, name, thrift.EXCEPTION, seqId)
	x.Write(ctx, oprot)
	oprot.WriteMessageEnd(ctx)
	oprot.Flush(ctx)
	return false, x
}
{{- end}}

{{- range .Functions}}
{{$FuncName := .GoName}}
{{$ProcessName := print ($ProcessorName | Unexport) $FuncName}}
{{$ArgType := .ArgType}}
{{$ResType := .ResType}}
type {{$ProcessorName | Unexport}}{{$FuncName}} struct {
	handler {{$ServiceName}}
}

func (p *{{$ProcessName}}) Process(ctx context.Context, seqId int32, iprot, oprot thrift.TProtocol) (success bool, err thrift.TException) {
	args := {{$ArgType.GoName}}{}
	var err2 error
	if err2 = args.Read(ctx, iprot); err2 != nil {
		iprot.ReadMessageEnd(ctx)
		{{- if not .Oneway}}
		x := thrift.NewTApplicationException(thrift.PROTOCOL_ERROR, err2.Error())
		oprot.WriteMessageBegin(ctx, "{{.Name}}", thrift.EXCEPTION, seqId)
		x.Write(ctx, oprot)
		oprot.WriteMessageEnd(ctx)
		oprot.Flush(ctx)
		{{- end}}
		return false, thrift.WrapTException(err2)
	}

	iprot.ReadMessageEnd(ctx)
	{{- if .Oneway}}
	if err2 = p.handler.{{$FuncName}}(ctx {{- range .Arguments}}, args.{{($ArgType.Field .Name).GoName}}{{- end}}); err2 != nil {
		return true, thrift.WrapTException(err2)
	}
	return true, nil
	{{- else}}
	result := {{$ResType.GoName}}{}
		{{- if .Void}}
	if err2 = p.handler.{{$FuncName}}(ctx {{- range .Arguments}}, args.{{($ArgType.Field .Name).GoName}}{{- end}}); err2 != nil {
		{{- else}}
	var retval {{.ResponseGoTypeName}}
	if retval, err2 = p.handler.{{$FuncName}}(ctx {{- range .Arguments}}, args.{{($ArgType.Field .Name).GoName}}{{- end}}); err2 != nil {
		{{- end}}{{/* if .Void */}}

		{{- if .Throws}}
		switch v := err2.(type) {
		{{- range .Throws}}
		case {{.GoTypeName}}:
			result.{{($ResType.Field .Name).GoName}} = v
		{{- end}}
		default:
			x := thrift.NewTApplicationException(thrift.INTERNAL_ERROR, "Internal error processing {{.Name}}: "+err2.Error())
			oprot.WriteMessageBegin(ctx, "{{.Name}}", thrift.EXCEPTION, seqId)
			x.Write(ctx, oprot)
			oprot.WriteMessageEnd(ctx)
			oprot.Flush(ctx)
			return true, thrift.WrapTException(err2)
		}
		{{- else}}
		x := thrift.NewTApplicationException(thrift.INTERNAL_ERROR, "Internal error processing {{.Name}}: "+err2.Error())
		oprot.WriteMessageBegin(ctx, "{{.Name}}", thrift.EXCEPTION, seqId)
		x.Write(ctx, oprot)
		oprot.WriteMessageEnd(ctx)
		oprot.Flush(ctx)
		return true, thrift.WrapTException(err2)
		{{- end}}{{/* if .Throws */}}
	{{- if not .Void}}
	} else {
		{{- with $rt := (index $ResType.Fields 0)}}
		result.Success = {{if and (NeedRedirect $rt.Field) (IsBaseType $rt.Type)}}&{{end}}retval
		{{- end}}
	{{- end}}
	}
	if err2 = oprot.WriteMessageBegin(ctx, "{{.Name}}", thrift.REPLY, seqId); err2 != nil {
		goto reply_err
	}
	if err2 = result.Write(ctx, oprot); err == nil && err2 != nil {
		goto reply_err
	}
	if err2 = oprot.WriteMessageEnd(ctx); err == nil && err2 != nil {
		goto reply_err
	}
	if err2 = oprot.Flush(ctx); err == nil && err2 != nil {
		goto reply_err
	}
	if err != nil {
		return
	}
	return true, err
reply_err:
	x := thrift.NewTApplicationException(thrift.INTERNAL_ERROR, "Internal error on response {{.Name}}: "+err2.Error())
	oprot.WriteMessageBegin(ctx, "{{.Name}}", thrift.EXCEPTION, seqId)
	x.Write(ctx, oprot)
	oprot.WriteMessageEnd(ctx)
	oprot.Flush(ctx)
	err = thrift.WrapTException(err2)
	return
	{{- end}}{{/* if .Oneway */}}
}
{{- end}}{{/* range .Functions */}}

{{- range .Functions}}
{{$ArgsType := .ArgType}}
{{template "StructLike" $ArgsType}}
{{- if not .Oneway}}
	{{$ResType := .ResType}}
	{{template "StructLike" $ResType}}
{{- end}}
{{- end}}{{/* range .Functions */}}
{{- end}}{{/* define "Processor" */}}
`
