// src/app/core/services/mount.ts
import { inject, Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { map } from 'rxjs/operators';
import { Observable } from 'rxjs';

export interface MountView {
  id: string;
  diskPath: string;
  name?: string;
}

@Injectable({ providedIn: 'root' })
export class MountsService {
  private http = inject(HttpClient);
  private base = '/api';

  getAll(): Observable<MountView[]> {
    return this.http.get<MountView[]>(`${this.base}/mounts`, {
      params: { t: Date.now().toString() }
    }).pipe(map(list => Array.isArray(list) ? list : []));
  }
}
