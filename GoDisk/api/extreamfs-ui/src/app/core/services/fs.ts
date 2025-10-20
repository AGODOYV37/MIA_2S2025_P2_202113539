import { inject, Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

export interface FindRes {
  ruta: string;
  name: string;
  items: string[]; 
}

@Injectable({ providedIn: 'root' })
export class FsService {
  private http = inject(HttpClient);
  private base = '/api/fs';

  find(ruta: string, name = '*'): Observable<FindRes> {
    return this.http.get<FindRes>(`${this.base}/find`, {
      params: { ruta, name, t: Date.now().toString() }
    });
    
  }
}
