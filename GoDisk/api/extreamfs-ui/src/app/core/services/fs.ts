import { inject, Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { LSReport } from './reports';

export interface FsFindResp {
  ruta: string;
  dirs: string[];
  files: string[];
}

@Injectable({ providedIn: 'root' })
export class FsService {
  private http = inject(HttpClient);
  private base = '/api/fs';
  private baseReports = '/api/reports';

  ls(id: string, ruta: string = '/'): Observable<LSReport> {
    return this.http.get<LSReport>(`${this.base}/ls`, {
      params: { id, ruta, t: Date.now().toString() }
    });
  }

  findList(id: string, ruta: string = '/'): Observable<FsFindResp> {
    return this.http.get<FsFindResp>(`${this.base}/find`, {
      params: { id, ruta, t: Date.now().toString() }
    });
  }


  fileText(id: string, ruta: string): Observable<string> {
    return this.http.get(`${this.baseReports}/file`, {
      params: { id, ruta, t: Date.now().toString() },
      responseType: 'text'
    });
  }
}
